# Starting the Server

This document describes how to start the Qwelli gRPC server.

## Prerequisites

- Go 1.21 or later installed
- All dependencies installed (run `go mod download` if needed)
- Protobuf code generated (see [protobuf.md](./protobuf.md))

## Quick Start

### Run the Server

From the project root directory:

```bash
go run cmd/qwellid/main.go
```

Or build and run:

```bash
go build ./cmd/qwellid
./qwellid
```

On Windows:

```powershell
go build ./cmd/qwellid
.\qwellid.exe
```

## Server Configuration

The server runs a gRPC server on **port 50051** by default.

**Features:**

- gRPC reflection enabled (allows tools like `grpcurl` to discover services)
- SearchService registered and ready to handle requests

When started successfully, you'll see:

```
Qwelli gRPC server running on :50051
```

## Available Services

- **SearchService** - Provides search functionality
  - RPC: `Search(SearchRequest) returns (SearchResponse)`

## Testing the Server

### Using grpcurl

Install `grpcurl`:

**macOS:**

```bash
brew install grpcurl
```

**Windows:**

Option 1 - Download binary (Recommended):

1. Download the latest Windows binary from: https://github.com/fullstorydev/grpcurl/releases
2. Extract `grpcurl.exe` to a directory in your PATH (e.g., `C:\Windows\System32` or add to your user PATH)

Option 2 - Using Scoop (if you have Scoop installed):

```powershell
scoop install grpcurl
```

Option 3 - Using Go (if you have Go installed):

```powershell
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

**Linux:**

```bash
# Download from releases or use package manager
wget https://github.com/fullstorydev/grpcurl/releases/latest/download/grpcurl_linux_x86_64.tar.gz
tar -xzf grpcurl_linux_x86_64.tar.gz
sudo mv grpcurl /usr/local/bin/
```

List services:

**PowerShell:**

```powershell
grpcurl -plaintext localhost:50051 list
```

**Bash:**

```bash
grpcurl -plaintext localhost:50051 list
```

Call the Search service:

**PowerShell:**

```powershell
# Option 1: Escape double quotes
grpcurl -plaintext -d "{\"query\": \"test\"}" localhost:50051 qwelli.v1.SearchService/Search

# Option 2: Use single quotes with escaped inner quotes (PowerShell 7+)
grpcurl -plaintext -d '{"query": "test"}' localhost:50051 qwelli.v1.SearchService/Search

# Option 3: Use a variable (recommended for complex JSON)
$json = '{"query": "test"}'
grpcurl -plaintext -d $json localhost:50051 qwelli.v1.SearchService/Search
```

**Bash:**

```bash
grpcurl -plaintext -d '{"query": "test"}' localhost:50051 qwelli.v1.SearchService/Search
```

### Using a gRPC Client

Create a Go client or use any gRPC client library to connect to `localhost:50051`.

## Troubleshooting

### Port Already in Use

If you see an error like `bind: address already in use`:

1. Find the process using port 50051:

   ```bash
   # Windows
   netstat -ano | findstr :50051

   # macOS/Linux
   lsof -i :50051
   ```

2. Stop the process or change the port in `cmd/qwellid/main.go`:
   ```go
   lis, err := net.Listen("tcp", ":50052")  // Change to different port
   ```

### Missing Dependencies

If you get import errors:

```bash
go mod download
go mod tidy
```

### Protobuf Code Not Generated

Ensure you've generated the protobuf code first:

```bash
protoc --go_out=. --go_opt=module=qwelli --go-grpc_out=. --go-grpc_opt=module=qwelli api/proto/search.proto
```

See [protobuf.md](./protobuf.md) for detailed instructions.

## Stopping the Server

Press `Ctrl+C` in the terminal where the server is running.

## Building for Production

Build an optimized binary:

```bash
go build -o qwellid ./cmd/qwellid
```

Build for different platforms:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o qwellid-linux ./cmd/qwellid

# Windows
GOOS=windows GOARCH=amd64 go build -o qwellid.exe ./cmd/qwellid

# macOS
GOOS=darwin GOARCH=amd64 go build -o qwellid-macos ./cmd/qwellid
```

## Environment Variables

Currently, the server uses hardcoded configuration. To make it configurable, you can:

1. Use environment variables:

   ```go
   port := os.Getenv("PORT")
   if port == "" {
       port = "50051"
   }
   ```

2. Use command-line flags with a package like `flag` or `cobra`

3. Use a config file (JSON, YAML, etc.)

## Next Steps

- Implement additional gRPC services
- Add authentication/authorization
- Add logging and monitoring
- Configure TLS/SSL for secure connections
- Add health checks and graceful shutdown
