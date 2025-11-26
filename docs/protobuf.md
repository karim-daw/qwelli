# Protobuf Code Generation

This document describes how to generate Go code from Protocol Buffer (protobuf) definitions in this project.

## Prerequisites

### 1. Install Protocol Buffers Compiler

Download and install `protoc` from the official releases:

- **Download**: https://github.com/protocolbuffers/protobuf/releases
- **Package Managers**:
  - Windows: `choco install protoc`
  - macOS: `brew install protobuf`
  - Linux: `apt-get install protobuf-compiler` or `yum install protobuf-compiler`

### 2. Install Go Protobuf Plugins

Install the required Go plugins for protobuf code generation:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Important**: Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH` so `protoc` can find these plugins.

Verify installation:

```bash
protoc-gen-go --version
protoc-gen-go-grpc --version
```

### 3. Install Go Dependencies

Add the required protobuf and gRPC dependencies to your project:

```bash
go get google.golang.org/protobuf/proto
go get google.golang.org/grpc
```

## Code Generation Command

### Standard Command

From the project root directory, run:

**PowerShell:**

```powershell
protoc --go_out=. --go_opt=module=github.com/karim-daw/qwelli --go-grpc_out=. --go-grpc_opt=module=github.com/karim-daw/qwelli api/proto/search.proto
```

**Bash:**

```bash
protoc --go_out=. --go_opt=module=github.com/karim-daw/qwelli --go-grpc_out=. --go-grpc_opt=module=github.com/karim-daw/qwelli api/proto/search.proto
```

### Command Breakdown

- `--go_out=.` - Output directory for Go message types (current directory)
- `--go_opt=module=github.com/karim-daw/qwelli` - Module name to use for import paths (matches `go.mod`)
- `--go-grpc_out=.` - Output directory for gRPC service code
- `--go-grpc_opt=module=github.com/karim-daw/qwelli` - Module name for gRPC imports
- `api/proto/search.proto` - Input proto file

### Generated Files

The command generates the following files in `api/gen/go/qwelli/v1/`:

- `search.pb.go` - Go structs for `SearchRequest` and `SearchResponse` messages
- `search_grpc.pb.go` - gRPC service interface and client/server code

## Project Structure

```
qwelli/
├── api/
│   ├── proto/                    # Protocol buffer definitions
│   │   └── search.proto
│   └── gen/
│       └── go/                   # Generated Go code (do not edit)
│           └── qwelli/
│               └── v1/
│                   ├── search.pb.go
│                   └── search_grpc.pb.go
└── ...
```

## Proto File Configuration

The `go_package` option in the proto file determines the import path:

```proto
option go_package = "github.com/karim-daw/qwelli/api/gen/go/qwelli/v1;v1;";
```

This means:

- Import path: `github.com/karim-daw/qwelli/api/gen/go/qwelli/v1`
- Package name: `v1`
- Generated files location: `api/gen/go/qwelli/v1/`

## Usage in Go Code

Import the generated code using:

```go
import (
    pb "github.com/karim-daw/qwelli/api/gen/go/qwelli/v1"
)
```

Example usage:

```go
// Create a request
req := &pb.SearchRequest{
    Query: "example query",
}

// Use the gRPC service
service := pb.NewSearchServiceClient(conn)
response, err := service.Search(ctx, req)
```

## Regenerating Code

After modifying `.proto` files:

1. Run the protoc command again
2. The generated files will be overwritten
3. Rebuild your Go code: `go build ./...`

## Troubleshooting

### "No such file or directory" error

The output directory structure is created automatically by protoc based on the `go_package` option. If you encounter issues, ensure the proto file's `go_package` matches your module name.

### "protoc-gen-go: program not found" error

1. Verify plugins are installed: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
2. Check your PATH includes `$GOPATH/bin` or `$HOME/go/bin`
3. On Windows, you may need to restart your terminal after adding to PATH

### Generated files in wrong location

- Ensure `go_package` option in proto file matches your module name (from `go.mod`)
- Use `module=github.com/karim-daw/qwelli` option (not `paths=source_relative`) to respect `go_package` paths
- Verify your `go.mod` module name matches the prefix in `go_package`

### Import errors in Go code

- Ensure the import path matches your module name + the path from `go_package`
- If module is `github.com/karim-daw/qwelli` and `go_package` is `github.com/karim-daw/qwelli/api/gen/go/qwelli/v1`, import as `github.com/karim-daw/qwelli/api/gen/go/qwelli/v1`
- Run `go mod tidy` to ensure dependencies are correct

## Adding New Proto Files

1. Create a new `.proto` file in `api/proto/`
2. Set the `go_package` option to match the pattern: `github.com/karim-daw/qwelli/api/gen/go/qwelli/v1;v1;`
3. Run the protoc command, including the new file:
   ```bash
   protoc --go_out=. --go_opt=module=github.com/karim-daw/qwelli --go-grpc_out=. --go-grpc_opt=module=github.com/karim-daw/qwelli api/proto/*.proto
   ```

## References

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [protoc-gen-go Documentation](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go)
