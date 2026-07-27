# AnyDesk Killer Remote

V2 uses an n8n form to enqueue an allowlisted `KILL_ANYDESK` command in a private Redis list. The Windows agent consumes it over an SSH local tunnel; Redis is never exposed publicly.

## Contents

- `n8n/Kill AnyDesk V2.json`: portable n8n workflow export.
- `agent/kill_anydesk_agent.pyw`: Windows agent.
- `agent/config.example.json`: safe configuration template.
- `agent/requirements.txt`: Python dependencies.
- `agent-go/`: Go implementation of the Windows agent (recommended for deployment).
- `.github/workflows/agent-go.yml`: CI for every push/PR; versioned Releases only for `v*` tags.

## Install the Go agent (recommended)

1. Create a restricted SSH user on the VPS and generate a dedicated key pair.
2. Copy the private key and a verified `known_hosts` file to the computer. Never leave `knownHosts` empty.
3. Copy `agent/config.example.json` to a local `config.json`, set the `machineId`, and keep it out of Git.
4. Download `anydesk-killer-agent-windows-amd64.zip` from a [GitHub Release](https://github.com/vitorhugo-dotnet/anydesk-killer-remote/releases), extract it, then run:

   ```powershell
   .\anydesk-killer-agent-windows-amd64.exe --config C:\AnyDeskKiller\config.json
   ```

The executable accepts only an unexpired, version-1 `KILL_ANYDESK` envelope targeted to its configured machine ID. There is no generic shell or remote-command execution path.

To publish a new binary, push a tag such as `v1.0.0`. The Release includes a ZIP and its SHA-256 checksum. Normal pushes and pull requests only run validation/build and upload a temporary Actions artifact.

## Python MVP agent

1. Create a restricted SSH user on the VPS and generate a dedicated key pair.
2. Copy the private key and a verified `known_hosts` file to the computer. Never use `known_hosts: null`.
3. Create `agent/config.json` from the example and set the local `machineId`.
4. Install dependencies with `py -m pip install -r agent/requirements.txt`.
5. Start it with `pyw agent/kill_anydesk_agent.pyw --config agent/config.json`.

The script has no shell or remote-command execution path. It accepts only an unexpired, version-1 `KILL_ANYDESK` envelope targeted to its own machine ID.

## n8n setup

The workflow exists in n8n as **Kill AnyDesk V2** and is intentionally unpublished. Before publishing, create the `Redis private VPS` credential in n8n pointing to the Redis endpoint reachable from the n8n container/host. Keep Redis bound to loopback or a private Docker network; do not publish port 6379.

Then attach that credential to **LPUSH na fila privada**, test each destination, and publish only after an agent has connected successfully.

## Workflow Git Sync

The n8n Data Table **Workflow Git Sync** holds one row per workflow destination. A workflow may have several rows, so it can be mirrored to more than one repository. Only repositories under `vitorhugo-dotnet/*` are permitted.

Required mapping columns are:

- `workflowId`, `workflowName`, `repository`, `branch`, and `filePath` identify the destination.
- `enabled` controls whether the mapping participates in sync.
- `status`, the two last hashes, and the conflict fields are managed by the sync workflows.

The synchronizer is deliberately conflict-safe: if both GitHub and n8n changed since the last baseline, it changes neither side, marks `CONFLICT`, and sends an e-mail. Resolve it only through the **Workflow Sync — Resolve Conflict** form with `N8N_WINS` or `GITHUB_WINS`.

Before enabling the workers, create an n8n API key in **Settings → n8n API** and save it as a least-privilege `n8n API` credential in n8n. Do not put the key in a workflow export or repository.
