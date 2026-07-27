from __future__ import annotations

import argparse
import asyncio
import json
import logging
import os
import subprocess
from dataclasses import dataclass
from datetime import UTC, datetime
from logging.handlers import RotatingFileHandler
from pathlib import Path
from typing import Any
from uuid import UUID

import asyncssh
import psutil
from redis.asyncio import Redis

MAX_MESSAGE_BYTES = 8 * 1024
ALLOWED_ACTION = "KILL_ANYDESK"


@dataclass(frozen=True)
class Config:
    machine_id: str
    ssh_host: str
    ssh_port: int
    ssh_username: str
    ssh_client_key: Path
    ssh_known_hosts: Path
    redis_remote_host: str
    redis_remote_port: int
    log_file: Path

    @property
    def command_queue(self) -> str:
        return f"remote-agent:commands:{self.machine_id}"


def load_config(path: Path) -> Config:
    raw = json.loads(path.read_text(encoding="utf-8"))
    ssh = raw["ssh"]
    redis = raw["redis"]
    config = Config(
        machine_id=raw["machineId"],
        ssh_host=ssh["host"],
        ssh_port=int(ssh.get("port", 22)),
        ssh_username=ssh["username"],
        ssh_client_key=Path(ssh["clientKey"]),
        ssh_known_hosts=Path(ssh["knownHosts"]),
        redis_remote_host=redis.get("remoteHost", "127.0.0.1"),
        redis_remote_port=int(redis.get("remotePort", 6379)),
        log_file=Path(raw.get("logFile", "logs/anydesk-agent.log")),
    )
    if not config.machine_id or not config.ssh_host or not config.ssh_username:
        raise ValueError("machineId and SSH host/username are required")
    if not config.ssh_client_key.is_file():
        raise ValueError("SSH clientKey was not found")
    if not config.ssh_known_hosts.is_file():
        raise ValueError("SSH knownHosts was not found")
    return config


def configure_logging(log_file: Path) -> logging.Logger:
    log_file.parent.mkdir(parents=True, exist_ok=True)
    logger = logging.getLogger("anydesk_agent")
    logger.setLevel(logging.INFO)
    handler = RotatingFileHandler(log_file, maxBytes=1_000_000, backupCount=3, encoding="utf-8")
    handler.setFormatter(logging.Formatter("%(asctime)s %(levelname)s %(message)s"))
    logger.addHandler(handler)
    return logger


def parse_utc(value: Any) -> datetime:
    if not isinstance(value, str):
        raise ValueError("timestamp must be a string")
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None or parsed.utcoffset() != UTC.utcoffset(parsed):
        raise ValueError("timestamp must be UTC")
    return parsed


def validate_command(raw: str, machine_id: str) -> dict[str, Any]:
    if len(raw.encode("utf-8")) >= MAX_MESSAGE_BYTES:
        raise ValueError("message is too large")

    command = json.loads(raw)
    if not isinstance(command, dict):
        raise ValueError("message must be an object")
    if command.get("version") != 1:
        raise ValueError("unsupported version")
    UUID(command["commandId"])
    UUID(command["correlationId"])
    if command.get("target") != machine_id:
        raise ValueError("message target does not match this machine")
    if command.get("action") != ALLOWED_ACTION:
        raise ValueError("action is not allowed")
    args = command.get("args")
    if not isinstance(args, dict) or set(args) - {"reopenAnyDesk"}:
        raise ValueError("arguments are not allowed for this action")
    if "reopenAnyDesk" in args and not isinstance(args["reopenAnyDesk"], bool):
        raise ValueError("reopenAnyDesk must be a boolean")

    requested_at = parse_utc(command["requestedAt"])
    expires_at = parse_utc(command["expiresAt"])
    if expires_at <= requested_at or expires_at <= datetime.now(UTC):
        raise ValueError("message has expired")
    return command


def kill_anydesk() -> dict[str, int]:
    matched: list[psutil.Process] = []
    for process in psutil.process_iter(["name"]):
        try:
            if (process.info["name"] or "").lower() == "anydesk.exe":
                matched.append(process)
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue

    for process in matched:
        try:
            process.terminate()
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue

    _, alive = psutil.wait_procs(matched, timeout=5)
    for process in alive:
        try:
            process.kill()
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue

    return {"matched": len(matched), "forceKilled": len(alive)}


def reopen_anydesk() -> bool:
    candidates = [
        Path(os.environ.get("ProgramFiles", r"C:\\Program Files")) / "AnyDesk" / "AnyDesk.exe",
        Path(os.environ.get("ProgramFiles(x86)", r"C:\\Program Files (x86)")) / "AnyDesk" / "AnyDesk.exe",
        Path(os.environ.get("LOCALAPPDATA", "")) / "AnyDesk" / "AnyDesk.exe",
    ]
    for executable in candidates:
        if executable.is_file():
            subprocess.Popen([str(executable)], close_fds=True)
            return True
    return False


async def publish(redis: Redis, queue: str, payload: dict[str, Any]) -> None:
    await redis.lpush(queue, json.dumps(payload, separators=(",", ":")))


async def consume(config: Config, logger: logging.Logger) -> None:
    delay = 1
    while True:
        redis: Redis | None = None
        try:
            logger.info("Connecting SSH tunnel to %s:%s", config.ssh_host, config.ssh_port)
            async with asyncssh.connect(
                config.ssh_host,
                port=config.ssh_port,
                username=config.ssh_username,
                client_keys=[str(config.ssh_client_key)],
                known_hosts=str(config.ssh_known_hosts),
                keepalive_interval=30,
                keepalive_count_max=3,
            ) as connection:
                listener = await connection.forward_local_port(
                    "127.0.0.1", 0, config.redis_remote_host, config.redis_remote_port
                )
                redis = Redis(
                    host="127.0.0.1",
                    port=listener.get_port(),
                    decode_responses=True,
                    socket_keepalive=True,
                )
                await redis.ping()
                logger.info("Connected; waiting on %s", config.command_queue)
                delay = 1

                while True:
                    item = await redis.brpop(config.command_queue, timeout=30)
                    if item is None:
                        continue
                    _, raw = item
                    try:
                        command = validate_command(raw, config.machine_id)
                        outcome = kill_anydesk()
                        reopen_requested = command["args"].get("reopenAnyDesk", False)
                        outcome["reopenAttempted"] = reopen_requested and outcome["matched"] > 0
                        outcome["reopened"] = reopen_anydesk() if outcome["reopenAttempted"] else False
                        result = {
                            "commandId": command["commandId"],
                            "correlationId": command["correlationId"],
                            "target": config.machine_id,
                            "status": "SUCCEEDED",
                            "completedAt": datetime.now(UTC).isoformat().replace("+00:00", "Z"),
                            "outcome": outcome,
                        }
                        await publish(redis, f"remote-agent:results:{config.machine_id}", result)
                        logger.info("KILL_ANYDESK completed: matched=%s forceKilled=%s", outcome["matched"], outcome["forceKilled"])
                    except Exception as exc:
                        rejected = {
                            "target": config.machine_id,
                            "status": "REJECTED",
                            "reason": str(exc),
                            "receivedAt": datetime.now(UTC).isoformat().replace("+00:00", "Z"),
                            "raw": raw[:MAX_MESSAGE_BYTES],
                        }
                        await publish(redis, f"remote-agent:dead-letter:{config.machine_id}", rejected)
                        logger.warning("Command rejected: %s", exc)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.exception("Transport failure; reconnecting in %s seconds", delay)
            await asyncio.sleep(delay)
            delay = min(delay * 2, 60)
        finally:
            if redis is not None:
                await redis.aclose()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default="config.json")
    args = parser.parse_args()
    config = load_config(Path(args.config))
    logger = configure_logging(config.log_file)
    asyncio.run(consume(config, logger))


if __name__ == "__main__":
    main()
