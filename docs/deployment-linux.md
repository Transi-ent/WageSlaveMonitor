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
sudo cp wageslave-server /opt/wageslave/bin/
```

## 3) Environment

Create `/etc/wageslave.env`:

```bash
ADDR=:8080
DATA_DIR=/opt/wageslave/data
DB_PATH=/opt/wageslave/data/meta.db
AUTH_TOKEN=replace_with_strong_token
CONSOLE_PASSWORD=replace_with_console_password
DEFAULT_CAPTURE_INTERVAL_SECONDS=30
RETENTION_DAYS=14
```

## 4) systemd

Create `/etc/systemd/system/wageslave.service`:

```ini
[Unit]
Description=WageSlaveMonitor Server
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/wageslave.env
WorkingDirectory=/opt/wageslave
ExecStart=/opt/wageslave/bin/wageslave-server
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

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
