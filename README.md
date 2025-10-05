# UP Parser for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/uplang/go.svg)](https://pkg.go.dev/github.com/uplang/go)
[![Go Report Card](https://goreportcard.com/badge/github.com/uplang/go)](https://goreportcard.com/report/github.com/uplang/go)
[![CI](https://github.com/uplang/go/workflows/CI/badge.svg)](https://github.com/uplang/go/actions)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

Official Go implementation of the UP (Unified Properties) language parser.

📚 **[API Documentation](https://pkg.go.dev/github.com/uplang/go)** | 🧪 **[Test Status](https://github.com/uplang/go/actions)** | 📖 **[Specification](https://github.com/uplang/spec)**

> **Reference Implementation** - This is the canonical implementation of the UP specification.

## Features

- ✅ **Full UP Syntax Support** - Scalars, blocks, lists, tables, multiline strings
- ✅ **Type Annotations** - Parse and preserve type hints (`!int`, `!bool`, etc.)
- ✅ **Functional Design** - Immutable data structures, functional parsing patterns
- ✅ **Well-Tested** - Comprehensive test suite with edge cases
- ✅ **Zero Dependencies** - Pure Go library implementation
- ✅ **Performance** - Efficient single-pass parser

## Requirements

- Go 1.25 or later

## Installation

### As a Library

```bash
go get github.com/uplang/go
```

### CLI Tool

The UP CLI tool is available in the [tools repository](https://github.com/uplang/tools):

```bash
# Main CLI
go install github.com/uplang/tools/up@latest

# Additional tools
go install github.com/uplang/tools/language-server@latest
go install github.com/uplang/tools/repl@latest
```

## Quick Start

```go
package main

import (
    "strings"
    up "github.com/uplang/go"
)

func main() {
    parser := up.NewParser()
    doc, err := parser.ParseDocument(strings.NewReader(`
        name Alice
        age!int 30
        config {
          debug!bool true
        }
    `))

    if err != nil {
        panic(err)
    }

    // Access parsed values
    for _, node := range doc.Nodes {
        println(node.Key, "=", node.Value)
    }
}
```

**📖 For detailed examples and tutorials, see [QUICKSTART.md](QUICKSTART.md)**

## Documentation

- **[QUICKSTART.md](QUICKSTART.md)** - Getting started guide with examples
- **[DESIGN.md](DESIGN.md)** - Architecture and design decisions
- **[UP Specification](https://github.com/uplang/spec)** - Complete language specification

## API Overview

### Core Types

- **`Parser`** - Main parser with configurable options
- **`Document`** - Parsed document containing nodes
- **`Node`** - Key-value pair with optional type annotation
- **`Value`** - Interface for all value types (scalar, block, list, table)

### Basic Usage

```go
// Create parser
parser := up.NewParser()

// Parse from io.Reader
doc, err := parser.ParseDocument(reader)

// Access nodes
for _, node := range doc.Nodes {
    fmt.Printf("%s: %v\n", node.Key, node.Value)
}
```

**See [DESIGN.md](DESIGN.md) for complete API documentation and implementation details.**

## CLI Tool

```bash
# Parse and pretty-print
up parse -i config.up --pretty

# Validate syntax
up validate config.up

# Convert to JSON
up convert -i config.up -o config.json --format json
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

## Project Structure

```
go/
├── parser.go           # Core parser implementation
├── types.go            # Data structures
├── parser_test.go      # Comprehensive tests
├── cmd/
│   └── up/            # CLI tool
│       └── main.go
├── README.md          # This file
├── QUICKSTART.md      # Getting started guide
├── DESIGN.md          # Architecture documentation
└── LICENSE            # GNU GPLv3
```

## Contributing

Contributions are welcome! Please see the main [CONTRIBUTING.md](https://github.com/uplang/spec/blob/main/CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Links

- **[UP Language Specification](https://github.com/uplang/spec)** - Official language spec
- **[Syntax Reference](https://github.com/uplang/spec/blob/main/SYNTAX-REFERENCE.md)** - Quick syntax guide
- **[UP Namespaces](https://github.com/uplang/ns)** - Official namespace plugins

### Other Implementations

- **[Java](https://github.com/uplang/java)** - Modern Java 21+ with records and sealed types
- **[JavaScript/TypeScript](https://github.com/uplang/js)** - Browser and Node.js support
- **[Python](https://github.com/uplang/py)** - Pythonic implementation with dataclasses
- **[Rust](https://github.com/uplang/rust)** - Zero-cost abstractions and memory safety
- **[C](https://github.com/uplang/c)** - Portable C implementation

## Support

- **Issues**: [github.com/uplang/go/issues](https://github.com/uplang/go/issues)
- **Discussions**: [github.com/uplang/spec/discussions](https://github.com/uplang/spec/discussions)
