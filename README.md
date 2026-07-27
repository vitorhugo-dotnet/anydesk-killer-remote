# AnyDesk Killer Remote

V2 uses an n8n form to enqueue an allowlisted `KILL_ANYDESK` command in a private Redis list. The Windows agent consumes it over an SSH local tunnel; Redis is never exposed publicly.

## Contents

- `n8n/Kill AnyDesk V2.json`: portable n8n workflow export.
- `agent/kill_anydesk_agent.pyw`: Windows agent.
- `agent/config.example.json`: safe configuration template.
- `agent/requirements.txt`: Python dependencies.

## Install the agent

1. Create a restricted SSH user on the VPS and generate a dedicated key pair.
2. Copy the private key and a verified `known_hosts` file to the computer. Never use `known_hosts: null`.
3. Create `agent/config.json` from the example and set the local `machineId`.
4. Install dependencies with `py -m pip install -r agent/requirements.txt`.
5. Start it with `pyw agent/kill_anydesk_agent.pyw --config agent/config.json`.

The script has no shell or remote-command execution path. It accepts only an unexpired, version-1 `KILL_ANYDESK` envelope targeted to its own machine ID.

## n8n setup

The workflow exists in n8n as **Kill AnyDesk V2** and is intentionally unpublished. Before publishing, create the `Redis private VPS` credential in n8n pointing to the Redis endpoint reachable from the n8n container/host. Keep Redis bound to loopback or a private Docker network; do not publish port 6379.

Then attach that credential to **LPUSH na fila privada**, test each destination, and publish only after an agent has connected successfully.
