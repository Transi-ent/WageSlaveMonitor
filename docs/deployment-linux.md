# Deploy Server on Linux

## 1) Build

```bash
cd server
go build -o wageslave-server ./cmd/server
```

## 2) Runtime directories

```bash
sudo mkdir -p /opt/wageslave/data
sudo mkdir -p /opt/wageslave/bin
sudo mkdir -p /opt/wageslave/config
sudo cp wageslave-server /opt/wageslave/bin/
```

## 3) Configuration

Create `/opt/wageslave/config/config.json`:

```json
{
  "ADDR": ":8080",
  "DATA_DIR": "/opt/wageslave/data",
  "DB_PATH": "/opt/wageslave/data/meta.db",
  "AUTH_TOKEN": "replace_with_strong_token",
  "DEFAULT_CAPTURE_INTERVAL_SECONDS": 30,
  "RETENTION_DAYS": 14,
  "CONSOLE_AUTH_DISABLED": false
}
```

**Note:** Set `CONSOLE_AUTH_DISABLED` to `true` for initial testing (no login required). Set to `false` to enable password protection (default password is `123456`).

## 4) systemd

Create `/etc/systemd/system/wageslave.service`:

```ini
[Unit]
Description=WageSlaveMonitor Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/wageslave
ExecStart=/opt/wageslave/bin/wageslave-server
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

The server will automatically read `/opt/wageslave/config/config.json`.

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable wageslave
sudo systemctl start wageslave
sudo systemctl status wageslave
```

## 5) Verify

```bash
curl http://127.0.0.1:8080/healthz
```

Should return `ok`.

## Configuration Reference

| Field | Description |
|-------|-------------|
| `ADDR` | Listen address, e.g., `:8080` or `127.0.0.1:8080`. |
| `DATA_DIR` | Root directory for screenshots and SQLite DB. |
| `DB_PATH` | Full path to SQLite database file. |
| `AUTH_TOKEN` | Bearer token for client API authentication (optional). |
| `DEFAULT_CAPTURE_INTERVAL_SECONDS` | Default screenshot interval for new clients. |
| `RETENTION_DAYS` | Auto-delete screenshots older than N days. |
| `CONSOLE_AUTH_DISABLED` | `true` to disable login, `false` to require password. |
