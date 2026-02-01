# Qwelli

Local semantic file search using vector embeddings. Index your folders and find files by meaning, not keywords.

## Prerequisites

- Go 1.21+ (for building from source)
- PostgreSQL with pgvector extension
- Voyage AI API key

**OR** use Docker Compose (recommended - no manual setup required)

## Quick Start

### 1. Get Voyage AI API Key

1. Go to https://www.voyageai.com/
2. Sign up and create a new API key
3. Copy it

### 2. Configure `.env`

**Qwelli automatically loads `.env` files!** No need to manually export variables.

```bash
# Copy the example file
cp .env.example .env

# Edit .env and add your Voyage API key
nano .env  # Change VOYAGE_API_KEY=your_voyage_api_key_here
```

The `.env` file contains:
- `DATABASE_URL` - Automatically set for local docker-compose
- `VOYAGE_API_KEY` - Your API key (required)
- `VOYAGE_MODEL` - Model to use (default: voyage-multimodal-3)
- `PORT` - Server port (default: 8080)
- `ENABLE_RERANKER` - Enable reranking (default: true)

### 3. Build and Run

#### CLI Mode

```bash
# Build the binary
go build -o qwelli ./cmd/qwelli

# Use commands (no need to export env vars!)
./qwelli index ./my-folder
./qwelli search "query" --index ./my-folder
./qwelli list
./qwelli shell
```

**Note:** The CLI automatically loads `.env` from the current directory or parent directories. No manual `export` needed!

#### Web UI Mode

```bash
# Build with embedded web UI
./build-with-ui.sh  # Linux/Mac
build-with-ui.bat   # Windows

# Start the web server
./qwelli serve

# Open browser to http://localhost:8080
```

**Important:** The `shell` command requires an interactive terminal. Always use the built binary (`./qwelli`) instead of `go run` for reliable operation, especially for the interactive shell.

## Deployment

### Local Development with Docker Compose (Recommended)

The easiest way to get started:

```bash
# 1. Clone the repository
git clone <your-repo>
cd qwelli

# 2. Create .env file
cp .env.example .env

# 3. Edit .env and add your Voyage API key
nano .env  # or vim, code, etc.

# 4. Start everything (PostgreSQL + Qwelli)
docker-compose up

# Access at http://localhost:8080
```

The docker-compose setup includes:
- PostgreSQL 16 with pgvector extension
- Automatic database initialization
- Health checks and auto-restart
- All configuration via `.env` file

### Production Deployment

#### Option 1: Docker Compose with Managed PostgreSQL

Best for simple production deployments on any cloud or VPS.

**Step 1: Set up managed PostgreSQL**

*Azure Example:*
```bash
# Create PostgreSQL Flexible Server
az postgres flexible-server create \
  --name qwelli-db \
  --resource-group mygroup \
  --location eastus \
  --admin-user qwelli \
  --admin-password <secure-password> \
  --sku-name Standard_B2s \
  --tier Burstable \
  --storage-size 32

# Enable pgvector extension
az postgres flexible-server parameter set \
  --resource-group mygroup \
  --server-name qwelli-db \
  --name azure.extensions \
  --value VECTOR

# Allow your server's IP
az postgres flexible-server firewall-rule create \
  --resource-group mygroup \
  --name qwelli-db \
  --rule-name allow-my-server \
  --start-ip-address <your-server-ip> \
  --end-ip-address <your-server-ip>
```

*AWS RDS Example:*
```bash
# Create RDS PostgreSQL instance with pgvector
aws rds create-db-instance \
  --db-instance-identifier qwelli-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 16.1 \
  --master-username qwelli \
  --master-user-password <secure-password> \
  --allocated-storage 20 \
  --vpc-security-group-ids <your-sg-id>

# Note: pgvector must be enabled via parameter group
```

**Step 2: Deploy application**

On your VPS/VM:
```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Clone repository
git clone <your-repo>
cd qwelli

# Create production .env
cat > .env << 'EOF'
DATABASE_URL=postgresql://qwelli:password@qwelli-db.postgres.database.azure.com:5432/qwelli?sslmode=require
VOYAGE_API_KEY=your_voyage_api_key
VOYAGE_MODEL=voyage-multimodal-3
PORT=8080
ENABLE_RERANKER=true
EOF

# Start with production configuration
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli

# View logs
docker-compose logs -f qwelli
```

The production setup:
- Uses external managed PostgreSQL (no local postgres container)
- Runs with production resource limits
- Automatic health checks and restarts
- Can use pre-built Docker images

**Step 3: Update/Redeploy**

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose build qwelli
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli

# Or use pre-built image
docker pull <your-registry>/qwelli:latest
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d qwelli
```

#### Option 2: Manual Build and Deployment

For maximum control or when Docker isn't available:

```bash
# 1. Build binary
go build -o qwelli ./cmd/qwelli

# 2. Set environment variables
export DATABASE_URL="postgresql://user:pass@host:5432/qwelli?sslmode=require"
export VOYAGE_API_KEY="your_key"
export PORT=8080

# 3. Run
./qwelli serve

# Or as systemd service (recommended for production)
sudo systemctl enable qwelli
sudo systemctl start qwelli
```

Example systemd service file (`/etc/systemd/system/qwelli.service`):
```ini
[Unit]
Description=Qwelli Semantic Search Service
After=network.target

[Service]
Type=simple
User=qwelli
WorkingDirectory=/opt/qwelli
EnvironmentFile=/opt/qwelli/.env
ExecStart=/opt/qwelli/qwelli serve
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Cloud-Specific Guides

#### Azure
- **Database:** Azure Database for PostgreSQL Flexible Server
- **Compute:** Azure Container Instances or VM
- **Cost:** ~$50-100/month (dev), ~$200-400/month (prod)

#### AWS
- **Database:** RDS for PostgreSQL with pgvector
- **Compute:** ECS Fargate, EC2, or App Runner
- **Cost:** ~$40-80/month (dev), ~$150-300/month (prod)

#### DigitalOcean / Linode / Other VPS
- **Database:** Managed PostgreSQL or self-hosted
- **Compute:** Standard Droplet/VM
- **Cost:** ~$10-50/month (all-in-one)

### Scaling Considerations

**Single Server (0-10k requests/day)**
- Docker Compose on single VPS works great
- Managed PostgreSQL (Standard_B2s or db.t3.small)
- 2 CPU / 4GB RAM VM

**Medium Scale (10k-100k requests/day)**
- Multiple app instances behind load balancer
- Managed PostgreSQL (Standard_D2s or db.t3.medium)
- Docker Swarm or simple orchestration

**Large Scale (100k+ requests/day)**
- Consider Kubernetes for orchestration
- PostgreSQL with read replicas
- CDN for static assets
- Caching layer (Redis)

## Usage

### Commands

#### `init`

Setup configuration (API key, model).

```bash
qwelli init
```

#### `index <folder>`

Index all files in a folder.

```bash
qwelli index ~/Documents/project
```

#### `search <query>`

Search indexed files.

```bash
qwelli search "machine learning" --index ~/Documents/project
qwelli search "api docs" -i ~/Documents/project -t 10
```

Options:

- `--index, -i` - Path to indexed folder
- `--top, -t` - Number of results (default: 5)

#### `list`

Show all indexed folders.

```bash
qwelli list
```

#### `status`

Show index statistics.

```bash
qwelli status --index ~/Documents/project
```

#### `shell`

Interactive mode.

```bash
qwelli shell
```

#### `serve`

Start web UI server.

```bash
qwelli serve
qwelli serve --port 3000  # Custom port
```

Options:

- `--port, -p` - Port number (default: 8080)

### Interactive Shell

```
qwelli> init                    # Setup config
qwelli> index ./my-folder       # Index and set as current
qwelli> search "query"          # Search current index
qwelli> use ./other-folder      # Switch current index
qwelli> list                    # Show all indexes
qwelli> status                  # Show index stats
qwelli> model                   # Show current model
qwelli> model gpt-4             # Change model (requires re-index)
qwelli> clear                   # Clear screen
qwelli> help                    # Show all commands
qwelli> exit                    # Exit
```

### Supported File Types

Text files only: `.txt`, `.md`, `.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp`, `.rs`, `.rb`, `.php`, `.html`, `.css`, `.yaml`, `.yml`, `.toml`, `.sh`, `.proto`, `.graphql`

**Note:** SQL files (`.sql`) are excluded as they can be very large (database dumps) and aren't suitable for semantic search.

Files >500KB and hidden files/folders are skipped.

## Build

```bash
# Current platform
go build -o qwelli ./cmd/qwelli

# Windows
GOOS=windows GOARCH=amd64 go build -o qwelli.exe ./cmd/qwelli

# Linux
GOOS=linux GOARCH=amd64 go build -o qwelli-linux ./cmd/qwelli

# macOS
GOOS=darwin GOARCH=amd64 go build -o qwelli-macos ./cmd/qwelli
GOOS=darwin GOARCH=arm64 go build -o qwelli-macos-arm ./cmd/qwelli
```

For Windows with bundled DuckDB (no GCC required at runtime):

```bash
./build-windows.bat
```

## Testing

### Unit Tests

```bash
# All tests
go test ./...

# Database tests only
go test ./internal/db/... -v

# With coverage
go test ./... -cover
```

### Integration Tests (requires API key)

```bash
# Set .env or export variables
go test ./internal/engine/indexer/... -v
```

### Demo

```bash
# Run end-to-end demo
go run tests/demo/main.go
```

### Manual Testing

```bash
# Build
go build -o qwelli ./cmd/qwelli

# Initialize
./qwelli init

# Index test data
./qwelli index tests/demo/testdata

# Search
./qwelli search "hello" --index tests/demo/testdata
./qwelli search "machine learning" --index tests/demo/testdata

# List and status
./qwelli list
./qwelli status --index tests/demo/testdata
```

## Embedding Providers

Currently supported: **Voyage AI**

### Voyage AI Models

| Model                 | Dimension | Notes                      |
| --------------------- | --------- | -------------------------- |
| `voyage-multimodal-3` | 1024      | Multimodal (text + images) |
| `voyage-3`            | 1024      | Text-only                  |

### Configuration

All configuration is via environment variables (see `.env.example`):

```bash
DATABASE_URL=postgresql://...          # Required: PostgreSQL connection
VOYAGE_API_KEY=your_key                # Required: Voyage AI API key
VOYAGE_MODEL=voyage-multimodal-3       # Optional: Model to use
PORT=8080                              # Optional: Server port
ENABLE_RERANKER=true                   # Optional: Enable result reranking
```

## Project Structure

```
qwelli/
├── cmd/qwelli/                 # CLI entry point
├── internal/
│   ├── cli/                    # Commands (init, index, search, shell)
│   ├── config/                 # Environment-based configuration
│   ├── db/                     # PostgreSQL with pgvector
│   ├── engine/                 # Index & search orchestration
│   │   ├── chunker/            # Content chunking strategies
│   │   ├── embeddings/         # Embedding providers (Voyage AI)
│   │   ├── extraction/         # PDF, image processing
│   │   └── fileprocessor/      # File type detection
│   ├── server/                 # Web server & API
│   └── voyage/                 # Voyage AI client
├── web/                        # React frontend
├── docker-compose.yml          # Local development setup
├── docker-compose.prod.yml     # Production overrides
├── Dockerfile                  # Container image
└── .env.example                # Environment variables template
```

## Data Storage

All data is stored in PostgreSQL with pgvector extension:

- **Files & Chunks:** Indexed content and metadata
- **Embeddings:** 1024-dimensional vectors (voyage-multimodal-3)
- **Vector Index:** pgvector HNSW index for fast similarity search

Configuration is environment-based (no config files).

## How It Works

1. **Index:** Scan folder → Chunk files → Generate embeddings via Voyage AI → Store in PostgreSQL
2. **Search:** Embed query → pgvector HNSW search → Optional reranking → Return matches

Features:
- **Multimodal:** Handles both text and images with `voyage-multimodal-3`
- **Fast:** pgvector HNSW index for approximate nearest neighbor search
- **Accurate:** Optional reranking for improved results
- **Scalable:** PostgreSQL can handle millions of documents

## Cost

Using Voyage AI `voyage-multimodal-3`:

- ~$0.02 per 1,000 documents indexed
- ~$0.0001 per search query

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.
