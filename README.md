# Signal Sideband

A privacy-focused Signal Relay Agent using Go, Postgres, and Signal-CLI.

## Setup

1. **Environment Variables**:
   Copy `.env.example` to `.env` and fill in your details.
   ```bash
   cp .env.example .env
   ```

2. **Start Infrastructure**:
   ```bash
   docker-compose up -d
   ```
   This starts the `signal-cli-rest-api` container.

3. **Register Signal Number**:
   If you haven't registered your number yet, use the included tool:
   ```bash
   # Make sure you have Twilio credentials in .env
   go run cmd/register/main.go
   ```
   Follow the prompts to solve the captcha.

4. **Database Setup**:
   Ensure your Postgres database is running (e.g., via Supabase or local).
   Run the schema script:
   ```bash
   psql "postgres://user:pass@host:5432/db" -f migrations/schema.sql
   ```

## Running the Relay

```bash
go run main.go
```

The service will:
- Connect to Signal.
- Log incoming messages.
- Embed message content (currently mock).
- Store messages in Postgres with vector embeddings.
- Delete expired messages automatically (Reaper).

## Architecture

- **cmd/register**: Automation for Signal registration via Twilio.
- **pkg/signal**: WebSocket client for bbernhard/signal-cli-rest-api.
- **pkg/store**: Postgres storage with `pgvector` support.
- **pkg/ai**: Interface for embedding providers.
