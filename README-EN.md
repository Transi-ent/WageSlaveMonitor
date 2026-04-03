# WageSlaveMonitor

[中文说明](README.md)

## Introduction

As a humble wage slave, have you ever been bothered by your company's monitoring software? As the saying goes: Know your enemy, know yourself, and you shall never lose a battle. This project is exactly such a tool — its name is WageSlaveMonitor.

WageSlaveMonitor is a lightweight **Windows client + Linux server** system for **authorized** endpoint operations and audit scenarios. The Windows agent periodically captures **all connected displays**, uploads JPEG screenshots to the server, and **buffers locally when offline** until the network returns. The server stores images on disk with **SQLite** metadata, exposes a simple **HTTP API**, and provides a **web console** to browse clients grouped by ID and screenshots sorted by time. Remote **capture interval** can be adjusted from the server side.

**Important:** Use only where you have explicit legal authority and informed consent. This software must not be used for covert surveillance or unauthorized monitoring.

## Features

| Area | Capability |
|------|------------|
| **Multi-monitor capture** | On Windows, captures every active display in one cycle (JPEG, quality tunable in code). |
| **Remote interval** | Server stores per-client `capture_interval_seconds`; client polls `GET /api/v1/clients/{id}/config`. |
| **Offline resilience** | Failed uploads stay in a local file spool under `AGENT_DATA_DIR/spool` and flush when connectivity returns. |
| **Lightweight server** | Single Go binary: SQLite index + filesystem blobs; optional retention job (`RETENTION_DAYS`). |
| **Web console** | List clients, drill into timeline (newest first), preview images; form to update capture interval. |
| **Console login** | Default password `123456` (stored as bcrypt in SQLite on first run); change it after login via **Change password**. Cookie session; API still accepts `Authorization: Bearer` when `AUTH_TOKEN` is set. |
| **Windows service** | `install` / `uninstall` subcommands register `WageSlaveMonitorAgent` via `sc.exe` (run elevated). |
| **Ops** | `GET /healthz`, request logging, Linux `systemd` example in [docs/deployment-linux.md](docs/deployment-linux.md). |

For architecture and API details, see [docs/mvp-architecture.md](docs/mvp-architecture.md).

## Quick Start

### Prerequisites

- **Go** 1.21+ (project modules may pin a newer toolchain).
- **Server:** Linux or any OS for development (paths in docs assume Linux for production).
- **Client:** **Windows** (capture uses Windows APIs; non-Windows builds use a stub).

### Server (development)

```bash
cd server
go run ./cmd/server
```

Server configuration is read from `server/config/config.json` (relative to the `server/` directory).

Key configuration fields:

| Field | Description |
|-------|-------------|
| `ADDR` | Listen address (default `:8080`). |
| `DATA_DIR` | Root for screenshots and DB (default `./data`). |
| `DB_PATH` | SQLite path (default `./data/meta.db`). |
| `AUTH_TOKEN` | If set, client API requires `Authorization: Bearer <token>`. |
| `DEFAULT_CAPTURE_INTERVAL_SECONDS` | Default interval for new clients (default `30`). |
| `RETENTION_DAYS` | Delete screenshots older than this many days (default `14`). |
| `CONSOLE_AUTH_DISABLED` | Set to `true` to disable console login (default `true` for easy testing). |

To enable password protection, set `CONSOLE_AUTH_DISABLED` to `false` in the config file. The default password is `123456`.

**Production on Linux** (build, systemd, config file): follow [docs/deployment-linux.md](docs/deployment-linux.md).

### Client (Windows)

```powershell
cd client
$env:SERVER_BASE_URL = "http://YOUR_SERVER:8080"
$env:AUTH_TOKEN = "same-as-server-if-set"
$env:AGENT_DATA_DIR = ".\agent-data"
go run .\cmd\agent
```

The agent creates a stable `client-id.txt` under `AGENT_DATA_DIR` on first run.

**Windows service (elevated PowerShell or CMD)**

```powershell
cd client
go build -o WageSlaveAgent.exe .\cmd\agent
# Run install from the built binary path:
.\WageSlaveAgent.exe install
# Uninstall:
.\WageSlaveAgent.exe uninstall
```

Configure the service's environment (e.g. `SERVER_BASE_URL`, `AUTH_TOKEN`) via your deployment method or `sc.exe` config as needed.

### Web console

1. Open `http://YOUR_SERVER:8080/console/clients`.
2. If `CONSOLE_AUTH_DISABLED` is `false` in the server config, sign in with the initial password **`123456`**, then use **Change password** in the top bar to set a strong password (minimum 6 characters).
3. Click a client to view screenshots (newest first) and adjust **Capture interval** via the form.

**API (optional, for automation)**

- List clients: `GET /api/v1/clients` with `Authorization: Bearer <AUTH_TOKEN>` if configured.
- Same header for ingest and config endpoints; see [docs/mvp-architecture.md](docs/mvp-architecture.md).

### Verify health

```bash
curl http://127.0.0.1:8080/healthz
```

Expect response body: `ok`.

## Contributing & community

If this project helps you, consider giving it a **Star** on GitHub — it helps others discover the repo and motivates maintenance. **Fork** the repository to experiment, open issues for bugs or ideas, and submit pull requests if you improve docs or code. Contributions that reinforce lawful, transparent use are especially welcome.

## License

This project is released under the [MIT License](LICENSE).
