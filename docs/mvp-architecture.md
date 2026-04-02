# WageSlaveMonitor MVP Architecture

This MVP is for authorized endpoint operations/audit scenarios only.

## Components

- `client/cmd/agent`: Windows agent process.
- `server/cmd/server`: Linux server binary.
- `server/web/templates`: Web console pages.

## Data Flow

1. Agent captures all active displays each interval.
2. Agent writes each screenshot to local spool.
3. Agent uploads spooled screenshots to server.
4. Server stores image blob to disk and metadata to SQLite.
5. Console reads grouped/timeline data from SQLite and serves previews from disk.

## API

- `POST /api/v1/clients/{clientId}/screenshots`
- `GET /api/v1/clients/{clientId}/config`
- `PUT /api/v1/clients/{clientId}/config`
- `GET /api/v1/clients`
- `GET /api/v1/clients/{clientId}/screenshots?page=1`
- `GET /console/clients`
- `GET /console/clients/{clientId}`
- `GET|POST /console/login`, `GET|POST /console/change-password` (default console password `123456` until changed; stored in SQLite)

## Storage

- Metadata DB: `${DATA_DIR}/meta.db`
- Images: `${DATA_DIR}/screenshots/{clientId}/YYYY/MM/DD/*.jpg`

## Reliability

- Offline queue: local file spool under `${AGENT_DATA_DIR}/spool`.
- Retry behavior: flush queue every capture loop; stop on first failed upload and retry later.
- Retention: server background job removes screenshots older than `RETENTION_DAYS`.
