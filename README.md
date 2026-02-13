# Signal Sideband

A Signal intelligence dashboard — captures messages from a Signal group via signal-cli, stores them with vector embeddings, and provides search, digests, media gallery, knowledge graph, and contact management through a React UI.

## Architecture

- **Go backend** — WebSocket listener for signal-cli, REST API, background workers (media download, AI analysis, link previews, digest scheduling, knowledge graph extraction)
- **React frontend** — Vite + TypeScript + Tailwind, served by the Go binary
- **PostgreSQL + pgvector** — message storage with full-text and semantic search
- **signal-cli-rest-api** — bbernhard's Docker image for Signal protocol access

### Package layout

| Package | Purpose |
|---------|---------|
| `pkg/signal` | WebSocket client + REST API client for signal-cli |
| `pkg/store` | Postgres storage (messages, contacts, groups, attachments, URLs, digests, cerebro) |
| `pkg/api` | HTTP handlers, auth middleware, CORS |
| `pkg/ai` | Embedding providers (OpenAI, mock) |
| `pkg/llm` | LLM providers (xAI/Grok, Claude, OpenAI, Perplexity) |
| `pkg/digest` | Digest generation, daily insights, scheduling |
| `pkg/cerebro` | Knowledge graph extraction and enrichment |
| `pkg/media` | Attachment download, thumbnails, AI vision analysis |
| `pkg/extract` | URL extraction and link preview fetching |
| `web/` | React 19 + Vite 7 + pnpm frontend |

## Setup

1. Copy `.env.example` to `.env` and fill in credentials
2. Start signal-cli: `docker compose up -d signal-cli`
3. Run database migrations: `psql $DATABASE_URL -f migrations/schema.sql`
4. Build and run: `go build -o signal-sideband . && ./signal-sideband`

The frontend is built separately (`cd web && pnpm build`) and served from `web/dist/`.

## Configuration

### Environment variables

See `.env.example` for the full list. Key settings:

| Variable | Purpose |
|----------|---------|
| `SIGNAL_URL` | WebSocket endpoint for signal-cli |
| `SIGNAL_API_URL` | REST endpoint for signal-cli |
| `SIGNAL_NUMBER` | Registered Signal phone number |
| `FILTER_GROUP_ID` | Only capture messages from this group (find via `GET /api/groups`) |
| `LLM_PROVIDER` | LLM for digests/insights (`xai`, `claude`, `openai`) |
| `OPENAI_API_KEY` | Used for embeddings |
| `XAI_API_KEY` | Used for LLM + vision analysis |
| `GEMINI_API_KEY` | Used for picture-of-the-day generation |

### Group filtering

Set `FILTER_GROUP_ID` to restrict message capture to a single group. On startup, any existing messages not in that group are purged (including their media files). Messages from other groups and DMs are silently dropped.

### Contact aliases

The Settings page (`/settings`) lets you assign display names to senders. Aliases resolve throughout Dashboard and Search via the `useContacts` hook. Backend: `GET /api/contacts`, `PUT /api/contacts/{uuid}`.

## Deployment (Norn)

Deployed via [Norn](https://github.com/antiartificial/norn) with auto-deploy on push to `master`.

- **infraspec.yaml** — declares env vars, secrets, volumes, migrations, healthcheck
- **secrets.enc.yaml** — SOPS-encrypted secrets (AGE key)
- **Dockerfile** — multi-stage build (Go + Node/pnpm + Alpine)
- Version is embedded at build time via `-ldflags "-X main.version=<sha>"` and exposed at `GET /api/version` and `GET /health`

### Migrations

Migrations run on every deploy via `for f in migrations/*.sql`. All migration files use `IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` to be idempotent. The `norn` DB user must own all tables (transferred via `ALTER TABLE ... OWNER TO norn`).

## Documentation

- **[Signal Events](docs/EVENTS.md)** — what signal-cli events we capture, how we use them, and the privacy model

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check + version |
| GET | `/api/version` | Build version |
| GET | `/api/stats` | Dashboard stats |
| GET | `/api/messages` | Paginated messages (filters: group_id, sender_id, after, before, has_media) |
| GET | `/api/messages/search` | Full-text or semantic search |
| GET | `/api/contacts` | All known senders + contact info |
| PUT | `/api/contacts/{uuid}` | Set contact alias |
| GET | `/api/groups` | List groups |
| GET | `/api/digests` | Paginated digests |
| POST | `/api/digests/generate` | Generate a digest |
| GET | `/api/urls` | Paginated URLs |
| GET | `/api/media` | Paginated attachments |
| GET | `/api/media/search` | Search media by AI analysis |
| POST | `/api/insights/generate` | Generate daily insight |
| GET | `/api/cerebro/graph` | Knowledge graph |
| POST | `/api/cerebro/extract` | Trigger extraction |
