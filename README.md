# qwelli

A local file search engine built in Go using gRPC. Qwelli provides hybrid search capabilities combining vector and semantic search for efficient local file discovery.

## Quick Start

### Prerequisites

- Go 1.21 or later
- Protobuf code generated (see [docs/protobuf.md](./docs/protobuf.md))

### Run

```bash
go run cmd/qwellid/main.go
```

### Build

```bash
go build ./cmd/qwellid
```

## Documentation

- [Architecture & Design](./docs/ARCHITECTURE.md) - Project structure, design decisions, build plan, and TODO list
- [Protobuf Setup](./docs/protobuf.md) - Protocol buffer code generation
- [Server Guide](./docs/server.md) - Starting and testing the gRPC server
