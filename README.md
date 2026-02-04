# Docko

PDF document management system for households and small teams. Ingest documents from local directories and network shares (SMB/NFS), extract text with OCR fallback, auto-tag with AI, and search across tens of thousands of documents.

## Features

- **Web Upload**: Drag-and-drop PDF upload with bulk support
- **Inbox Watching**: Auto-import from watched local directories
- **Network Shares**: Import from SMB and NFS shares on schedule
- **Text Extraction**: Embedded text extraction with OCRmyPDF fallback
- **Full-Text Search**: PostgreSQL-powered search with tag, correspondent, and date filters
- **AI Tagging**: Auto-suggest tags and correspondents (OpenAI, Anthropic, Ollama)
- **Organization**: Tags and correspondents with merge support
- **PDF Viewer**: In-browser preview with download option
- **Dashboard**: Overview of document counts, queue health, and recent activity
- **Queue Management**: Monitor processing queues, retry failed jobs, view activity

## Quick Start (Development)

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Node.js (for Tailwind CSS)
- direnv (recommended)

### Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/docko.git
cd docko

# Start PostgreSQL and OCRmyPDF
docker compose up -d

# Copy environment example and configure
cp .envrc.example .envrc
# Edit .envrc with your settings (defaults work for docker-compose)
direnv allow

# Install Go dependencies
go mod download

# Install dev tools (templ, sqlc, air, tailwind)
make setup

# Start development server with hot reload
make dev
```

The app will be available at http://localhost:3000

### Default Credentials

- **Username**: `admin`
- **Password**: Set via `ADMIN_PASSWORD` in `.envrc` (default: `changeme123`)

## Production Deployment

### Prerequisites

- Docker and Docker Compose v2
- Domain name (optional, for SSL via reverse proxy)

### Step 1: Generate Secrets

```bash
# Session secret (required - used for session cookie HMAC)
export SESSION_SECRET=$(openssl rand -hex 32)
echo "SESSION_SECRET=$SESSION_SECRET"

# Credential encryption key (required for network sources - AES-256)
export CREDENTIAL_ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "CREDENTIAL_ENCRYPTION_KEY=$CREDENTIAL_ENCRYPTION_KEY"

# Database password
export POSTGRES_PASSWORD=$(openssl rand -hex 16)
echo "POSTGRES_PASSWORD=$POSTGRES_PASSWORD"

# Admin password (choose a strong password)
export ADMIN_PASSWORD="your-secure-password-here"
```

Save these values securely. You'll need them for configuration.

### Step 2: Configure Environment

Create a `.env` file in the project root:

```bash
# Required
DATABASE_URL=postgres://docko:YOUR_POSTGRES_PASSWORD@postgres:5432/docko?sslmode=disable
POSTGRES_PASSWORD=YOUR_POSTGRES_PASSWORD
ADMIN_PASSWORD=YOUR_ADMIN_PASSWORD
SESSION_SECRET=YOUR_SESSION_SECRET
CREDENTIAL_ENCRYPTION_KEY=YOUR_CREDENTIAL_KEY

# Recommended
SITE_URL=https://docko.example.com
LOG_LEVEL=INFO

# Optional: AI providers (configure at least one for AI features)
# OPENAI_API_KEY=sk-your-openai-key
# ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
# OLLAMA_URL=http://ollama:11434
```

### Step 3: Create Storage Directories

```bash
mkdir -p storage/ocr-input storage/ocr-output
```

### Step 4: Deploy

```bash
# Build and start all services
docker compose -f docker-compose.prod.yml up -d --build

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f app
```

### Step 5: Access the Application

Navigate to http://your-server:3000 (or configure a reverse proxy for SSL).

### Reverse Proxy (SSL)

Docko runs on HTTP. Use a reverse proxy like nginx or Caddy for SSL termination.

**Caddy (easiest - auto-SSL):**

```caddyfile
docko.example.com {
    reverse_proxy localhost:3000
}
```

**nginx:**

```nginx
server {
    listen 443 ssl;
    server_name docko.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string. Format: `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| `ADMIN_PASSWORD` | Admin user password (bcrypt hashed on startup) |
| `SESSION_SECRET` | HMAC secret for session cookies. Generate with `openssl rand -hex 32` |
| `POSTGRES_PASSWORD` | PostgreSQL password (for docker-compose.prod.yml) |

### Required for Network Sources

| Variable | Description |
|----------|-------------|
| `CREDENTIAL_ENCRYPTION_KEY` | AES-256 key for encrypting network source credentials. Generate with `openssl rand -hex 32` |

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP server port |
| `ENV` | `development` | Environment mode (`development` or `production`) |
| `LOG_LEVEL` | `INFO` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `SITE_NAME` | `Docko` | Site name for meta tags and page titles |
| `SITE_URL` | `http://localhost:3000` | Base URL for canonical links and OG tags |
| `DEFAULT_OG_IMAGE` | `/static/images/og-default.png` | Default OpenGraph image path |
| `STORAGE_PATH` | `./storage` | Root path for document storage |
| `INBOX_PATH` | - | Default inbox directory path (disabled if not set) |
| `INBOX_ERROR_SUBDIR` | `errors` | Subdirectory for files that fail processing |
| `INBOX_MAX_FILE_SIZE_MB` | `100` | Maximum file size in MB for inbox imports |
| `INBOX_SCAN_INTERVAL_MS` | `1000` | Directory scan interval in milliseconds |
| `SESSION_MAX_AGE` | `24` | Session max age in hours |

### AI Provider Configuration

Configure at least one provider to use AI features (tag/correspondent suggestions).

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | - | OpenAI API key (uses gpt-4o-mini model) |
| `ANTHROPIC_API_KEY` | - | Anthropic API key (uses Claude Haiku 4.5 model) |
| `OLLAMA_URL` | - | Ollama server URL (e.g., `http://localhost:11434`) |
| `OLLAMA_MODEL` | `llama3.2` | Ollama model name |

AI providers are tried in order: OpenAI -> Anthropic -> Ollama. Configure multiple for fallback.

See `.envrc.example` for complete configuration reference with detailed comments.

## Backup & Restore

### Database Backup

```bash
# Create backup
docker exec docko-postgres pg_dump -U docko docko > backup-$(date +%Y%m%d).sql

# Compressed backup (recommended for larger databases)
docker exec docko-postgres pg_dump -U docko docko | gzip > backup-$(date +%Y%m%d).sql.gz
```

### Database Restore

```bash
# From plain SQL
docker exec -i docko-postgres psql -U docko docko < backup.sql

# From compressed backup
gunzip -c backup.sql.gz | docker exec -i docko-postgres psql -U docko docko
```

### Document Storage Backup

```bash
# Backup storage volume
docker run --rm \
  -v docko-storage:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/storage-$(date +%Y%m%d).tar.gz -C /data .
```

### Document Storage Restore

```bash
# Restore storage volume
docker run --rm \
  -v docko-storage:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/storage-backup.tar.gz"
```

### Full Backup Script

Create `backup.sh`:

```bash
#!/bin/bash
set -e

DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="./backups/$DATE"
mkdir -p "$BACKUP_DIR"

echo "Starting backup to $BACKUP_DIR..."

# Database
echo "Backing up database..."
docker exec docko-postgres pg_dump -U docko docko | gzip > "$BACKUP_DIR/database.sql.gz"

# Storage
echo "Backing up document storage..."
docker run --rm \
  -v docko-storage:/data \
  -v "$BACKUP_DIR":/backup \
  alpine tar czf /backup/storage.tar.gz -C /data .

# Environment (optional - be careful with secrets)
# cp .env "$BACKUP_DIR/env.backup"

echo "Backup complete: $BACKUP_DIR"
ls -lh "$BACKUP_DIR"
```

Make it executable: `chmod +x backup.sh`

### Full Restore Script

Create `restore.sh`:

```bash
#!/bin/bash
set -e

if [ -z "$1" ]; then
    echo "Usage: ./restore.sh <backup-directory>"
    echo "Example: ./restore.sh ./backups/20260204-120000"
    exit 1
fi

BACKUP_DIR="$1"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "Error: Backup directory not found: $BACKUP_DIR"
    exit 1
fi

echo "Restoring from $BACKUP_DIR..."

# Stop app (keep postgres running)
docker compose -f docker-compose.prod.yml stop app

# Restore database
echo "Restoring database..."
gunzip -c "$BACKUP_DIR/database.sql.gz" | docker exec -i docko-postgres psql -U docko docko

# Restore storage
echo "Restoring document storage..."
docker run --rm \
  -v docko-storage:/data \
  -v "$BACKUP_DIR":/backup \
  alpine sh -c "cd /data && rm -rf * && tar xzf /backup/storage.tar.gz"

# Restart app
docker compose -f docker-compose.prod.yml start app

echo "Restore complete"
```

Make it executable: `chmod +x restore.sh`

## Upgrade Procedures

### Standard Upgrade

1. **Backup first:**

```bash
./backup.sh  # Or manual backup commands above
```

2. **Pull latest code:**

```bash
git pull origin main
```

3. **Check for configuration changes:**

```bash
# Compare your config with the example
diff .envrc .envrc.example
# Or for production
diff .env .envrc.example
```

4. **Add any new required environment variables.**

5. **Rebuild and restart:**

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

6. **Verify:**

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs app | tail -20
curl http://localhost:3000/health
```

### Database Migrations

Migrations run automatically on startup using Goose. If a migration fails:

1. Check logs: `docker compose -f docker-compose.prod.yml logs app`
2. Restore from backup if needed
3. Fix the issue and restart

### Rollback

If an upgrade causes issues:

1. Stop the app: `docker compose -f docker-compose.prod.yml stop app`
2. Restore from backup: `./restore.sh ./backups/YYYYMMDD-HHMMSS`
3. Checkout previous version: `git checkout <previous-commit>`
4. Rebuild: `docker compose -f docker-compose.prod.yml up -d --build`

## Troubleshooting

### Container Won't Start

**Check logs:**
```bash
docker compose -f docker-compose.prod.yml logs app
```

**Common issues:**

| Error | Solution |
|-------|----------|
| `connection refused` to postgres | Wait for postgres health check to pass, or check `DATABASE_URL` format |
| `missing required environment variable` | Ensure all required vars are set in `.env` |
| `permission denied` | Ensure storage directories exist and are writable |
| `migration failed` | Check database connectivity, restore from backup if needed |

### Health Check Failing

**Verify health endpoint:**
```bash
curl http://localhost:3000/health
# Should return: OK
```

**Check container health:**
```bash
docker inspect docko-app | grep -A 10 Health
```

**Check resource usage:**
```bash
docker stats docko-app
```

### Database Connection Issues

**Test connection:**
```bash
docker exec docko-postgres psql -U docko -d docko -c "SELECT 1"
```

**Check DATABASE_URL format:**
```
postgres://user:password@host:port/database?sslmode=disable
```

**Common mistakes:**
- Using `localhost` instead of `postgres` (container name) in Docker
- Missing `?sslmode=disable` for Docker connections
- Wrong port (default is 5432)

### OCR Not Working

**Check OCRmyPDF service:**
```bash
docker compose -f docker-compose.prod.yml logs ocrmypdf
```

**Verify directories exist:**
```bash
ls -la storage/ocr-input storage/ocr-output
```

**Manual test:**
```bash
# Copy a test PDF
cp test.pdf storage/ocr-input/

# Wait 30 seconds for processing
sleep 30

# Check output
ls storage/ocr-output/
```

**Check inotify is working:**
```bash
docker exec docko-ocrmypdf ls /input /output
```

### AI Tagging Not Working

**Check provider configuration in Settings > AI:**
- Verify the provider shows as "Available"
- Check API keys are set correctly

**Test providers:**
- **OpenAI**: Verify `OPENAI_API_KEY` is set and valid
- **Anthropic**: Verify `ANTHROPIC_API_KEY` is set and valid
- **Ollama**: Verify `OLLAMA_URL` points to a running Ollama server

**Check AI queue:**
- Go to Queues page in the UI
- Look for failed AI jobs with error messages

### Network Sources Not Syncing

**Check network source status:**
- Go to Settings > Network Sources
- Verify source is enabled
- Check last sync time and any error messages

**Test connectivity:**
- For SMB: Ensure port 445 is accessible
- For NFS: Ensure port 2049 is accessible
- Verify credentials are correct

**Check logs:**
```bash
docker compose -f docker-compose.prod.yml logs app | grep -i network
```

### High Memory Usage

**Check container stats:**
```bash
docker stats
```

**If app container exceeds limits:**
- Increase memory limit in `docker-compose.prod.yml`
- Check for memory leaks in logs

**Adjust resource limits:**
```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 1024M
```

### Logs Filling Disk

Docker logging is configured with rotation (10MB, 5 files). If logs are still growing:

```bash
# Check log sizes
docker system df -v

# Prune old containers and images
docker system prune -f

# Recreate containers to apply logging config
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d
```

### Slow Search Performance

**Check for missing indexes:**
```bash
docker exec docko-postgres psql -U docko -d docko -c "\di"
```

**Analyze table statistics:**
```bash
docker exec docko-postgres psql -U docko -d docko -c "ANALYZE documents;"
```

**Check search vector:**
```bash
docker exec docko-postgres psql -U docko -d docko -c "SELECT count(*) FROM documents WHERE search_vector IS NULL;"
```

## Development

### Project Structure

```
cmd/server/          Entry point and slog config
internal/
  ai/                AI providers (OpenAI, Anthropic, Ollama)
  auth/              Authentication service
  config/            Environment configuration
  ctxkeys/           Typed context keys
  database/          Database connection, migrations, sqlc
  document/          Document service
  handler/           HTTP handlers
  inbox/             Inbox watcher service
  meta/              SEO/OG metadata helpers
  middleware/        Echo middleware
  network/           Network source protocols (SMB, NFS)
  processing/        Document processing pipeline
  queue/             Job queue system
  storage/           Document storage service
  testutil/          Test helpers
templates/           Templ templates
  layouts/           Base layouts (base.templ, admin.templ, login.templ)
  pages/             Page templates
components/          templUI components (button, card, input, etc.)
static/              Static assets
  css/               Tailwind CSS (input.css, output.css)
  js/                JavaScript files
  images/            Static images
sqlc/                SQL queries and configuration
  queries/           SQL query files
  sqlc.yaml          SQLC configuration
assets/              templUI JavaScript files
```

### Key Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start with hot reload (main development workflow) |
| `make build` | Build production binary |
| `make test` | Run tests with race detection |
| `make lint` | Run golangci-lint and templ fmt |
| `make generate` | Regenerate templ + sqlc code |
| `make migrate` | Run database migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-status` | Show migration status |
| `make migrate-create NAME=xxx` | Create new migration |
| `make css-watch` | Watch Tailwind CSS (run in separate terminal) |
| `make setup` | Install development tools |
| `make clean` | Remove build artifacts |

### Development Workflow

1. Start PostgreSQL and OCRmyPDF: `docker compose up -d`
2. Start dev server: `make dev`
3. Edit code - server auto-reloads
4. Check `./tmp/air-combined.log` for compilation errors
5. Run tests: `make test`

### Adding Components

The project uses [templUI](https://templui.io/) for UI components:

```bash
# Install CLI (one-time)
go install github.com/templui/templui/cmd/templui@latest

# Add components
templui add button card input label

# Regenerate code
make generate
```

### Tech Stack

- **Backend**: Go 1.24+, Echo framework
- **Templates**: Templ
- **Frontend**: HTMX, Tailwind CSS, templUI components
- **Database**: PostgreSQL 16, sqlc for type-safe queries
- **OCR**: OCRmyPDF (Docker service)
- **Migrations**: Goose (embedded in binary)

## Security

### Production Checklist

- [ ] Change default `ADMIN_PASSWORD`
- [ ] Generate unique `SESSION_SECRET` (32+ chars)
- [ ] Generate unique `CREDENTIAL_ENCRYPTION_KEY` (32 chars)
- [ ] Use HTTPS via reverse proxy
- [ ] Restrict network access to PostgreSQL
- [ ] Enable firewall, only expose necessary ports
- [ ] Regular backups with offsite storage
- [ ] Keep dependencies updated

### Authentication

- Single admin user with bcrypt-hashed password
- Session cookies with HMAC signature
- Sessions stored in database with expiry

### Data Security

- Network source credentials encrypted with AES-256-GCM
- Document storage uses UUID-sharded paths
- No sensitive data in logs

## License

[Add your license here]
