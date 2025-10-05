# Quick Start Guide - Go

Get started with the UP Parser for Go in 5 minutes!

## Installation

```bash
# Install as a library
go get github.com/uplang/go

# Install CLI tool
go install github.com/uplang/go/cmd/up@latest
```

## Your First Program

Create `main.go`:

```go
package main

import (
    "fmt"
    "strings"
    up "github.com/uplang/go"
)

func main() {
    // Create a parser
    parser := up.NewParser()

    // Parse UP content
    input := `
name Alice
age!int 30
active!bool true
`

    doc, err := parser.ParseDocument(strings.NewReader(input))
    if err != nil {
        panic(err)
    }

    // Print all nodes
    for _, node := range doc.Nodes {
        fmt.Printf("%s = %v\n", node.Key, node.Value)
    }
}
```

Run it:

```bash
go run main.go
```

Output:
```
name = Alice
age = 30
active = true
```

## Common Use Cases

### 1. Parse Configuration Files

```go
package main

import (
    "os"
    up "github.com/uplang/go"
)

func main() {
    file, _ := os.Open("config.up")
    defer file.Close()

    parser := up.NewParser()
    doc, err := parser.ParseDocument(file)
    if err != nil {
        panic(err)
    }

    // Use the parsed configuration
    for _, node := range doc.Nodes {
        if block, ok := node.Value.(up.Block); ok {
            fmt.Printf("%s block:\n", node.Key)
            for k, v := range block {
                fmt.Printf("  %s = %v\n", k, v)
            }
        }
    }
}
```

### 2. Working with Blocks

```go
input := `
server {
  host localhost
  port!int 8080
}
`

doc, _ := parser.ParseDocument(strings.NewReader(input))

for _, node := range doc.Nodes {
    if node.Key == "server" {
        if block, ok := node.Value.(up.Block); ok {
            host := block["host"].(string)
            port := block["port"].(string)
            fmt.Printf("Server: %s:%s\n", host, port)
        }
    }
}
```

### 3. Working with Lists

```go
input := `tags [web, api, production]`

doc, _ := parser.ParseDocument(strings.NewReader(input))

for _, node := range doc.Nodes {
    if node.Key == "tags" {
        if list, ok := node.Value.(up.List); ok {
            for _, item := range list {
                fmt.Println("Tag:", item)
            }
        }
    }
}
```

### 4. Type Annotations

```go
doc, _ := parser.ParseDocument(strings.NewReader(`
port!int 8080
enabled!bool true
timeout!dur 30s
`))

for _, node := range doc.Nodes {
    if node.Type != "" {
        fmt.Printf("%s has type: %s\n", node.Key, node.Type)
    }
}
```

## Using the CLI

### Parse and Validate

```bash
# Parse a file
up parse -i config.up

# Pretty print
up parse -i config.up --pretty

# Validate syntax
up validate config.up
```

### Convert Formats

```bash
# Convert to JSON
up convert -i config.up -o config.json --format json

# Convert to YAML
up convert -i config.up -o config.yaml --format yaml
```

## Next Steps

- Read the [DESIGN.md](DESIGN.md) for implementation details
- Explore the [UP Specification](https://github.com/uplang/spec)
- Check out [example files](https://github.com/uplang/spec/tree/main/examples)
- Try [UP Namespaces](https://github.com/uplang/ns) for extended functionality

## Need Help?

- 📚 [Full Documentation](README.md)
- 💬 [Discussions](https://github.com/uplang/spec/discussions)
- 🐛 [Report Issues](https://github.com/uplang/go/issues)

