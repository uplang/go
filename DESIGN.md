# Design Documentation - Go Implementation

This document describes the architecture and design decisions of the Go UP parser implementation.

## Overview

The Go implementation serves as the **reference implementation** for the UP language. It prioritizes:

- **Correctness** - Strictly follows the specification
- **Simplicity** - Clean, readable code
- **Functional Design** - Immutable data structures, pure functions
- **Performance** - Efficient single-pass parsing
- **Zero Dependencies** - Only uses Go standard library

## Architecture

### Core Components

```
┌─────────────┐
│   Scanner   │  Reads lines, tracks line numbers
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Parser    │  Parses UP syntax into AST
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Document   │  Immutable parsed representation
└─────────────┘
```

### Package Structure

```
github.com/uplang/go/
├── parser.go           # Parser implementation
├── types.go            # Data structures
├── parser_test.go      # Tests
└── cmd/up/
    └── main.go         # CLI tool
```

## Data Structures

### Document

Represents a fully parsed UP document:

```go
type Document struct {
    Nodes []Node  // Ordered list of top-level nodes
}
```

**Design Rationale:**
- Slice preserves order (important for configuration)
- Simple structure, easy to iterate
- Immutable after parsing

### Node

Represents a key-value pair with optional type annotation:

```go
type Node struct {
    Key   string  // The key name
    Type  string  // Optional type annotation (e.g., "int", "bool")
    Value Value   // The parsed value
}
```

**Design Rationale:**
- Separates key from value for easy access
- Type is optional (empty string if not specified)
- Value is an interface to support multiple types

### Value Types

```go
type Value interface{}  // Marker interface

// Concrete types:
type Block map[string]Value     // Nested key-value pairs
type List []Value                // Ordered collection
type Table struct {              // Tabular data
    Columns []interface{}
    Rows    []interface{}
}
// Scalar values are stored as strings
```

**Design Rationale:**
- Interface allows heterogeneous collections
- `Block` uses map for O(1) lookup
- `List` preserves order
- Strings avoid type conversion errors
- Users perform type conversion as needed

## Parser Implementation

### Functional Parsing Pattern

The parser uses a functional approach with higher-order functions:

```go
type ParseFunc[T any] func(*Scanner, string) (T, error)

func (p *Parser) WithDedentFunc(fn func(string, int) string) *Parser
func (p *Parser) WithSkipEmptyLine(fn func(string) bool) *Parser
func (p *Parser) WithSkipComment(fn func(string) bool) *Parser
```

**Design Rationale:**
- Configurable behavior without inheritance
- Easy to test individual components
- Composable parsing functions
- Type-safe with generics (Go 1.18+)

### Single-Pass Parsing

The parser reads input once, never backtracking:

```
Input Stream → Scanner → Parse Line → Build Node → Add to Document
```

**Advantages:**
- Memory efficient
- Predictable performance O(n)
- Simple error handling
- Works with streams (doesn't require full input in memory)

### Error Handling

Errors include line numbers and context:

```go
return fmt.Errorf("line %d: %w", lineNum, err)
```

**Design Rationale:**
- User-friendly error messages
- Easy to locate problems in source files
- Errors wrap for context preservation

## Parsing Strategy

### Whitespace Handling

UP is **whitespace-delimited**. The parser:

1. Splits on first whitespace to get key
2. Takes remainder as value
3. Handles indentation for blocks

```go
func splitFirstWhitespace(str string) (string, string) {
    // Split on first space or tab
}
```

### Block Parsing

Blocks are parsed recursively:

```
server {        ← Detect opening brace
  host ...      ← Parse nested nodes recursively
  port ...      ← Continue until closing brace
}               ← Return Block value
```

**Design Rationale:**
- Natural recursive structure
- Each block is self-contained
- Stack-based nesting tracking

### List Parsing

Lists support two formats:

```up
# Inline
tags [item1, item2, item3]

# Multiline
tags [
  item1
  item2
]
```

**Implementation:**
- Detect `[` to start list
- If `]` on same line → inline mode
- Otherwise → multiline mode
- Parse items until `]`

### Multiline String Parsing

```up
description ```
Multi-line content
Preserves whitespace
```
```

**Implementation:**
- Detect triple backticks
- Capture all lines until closing backticks
- No escaping or processing (raw content)

## Type System

### Type Annotations

Syntax: `key!type value`

```go
// Parsed as:
Node{
    Key:   "port",
    Type:  "int",
    Value: "8080",
}
```

**Design Rationale:**
- Types are metadata, not enforced by parser
- Parser preserves type hints for consumers
- Validation is responsibility of application
- Flexible for schema validation layer

### Scalar Values

All scalar values are stored as strings:

```go
"8080"          // Even for port!int 8080
"true"          // Even for active!bool true
"30s"           // Even for timeout!dur 30s
```

**Design Rationale:**
- Avoids type conversion errors in parser
- Preserves exact input representation
- Users convert based on type annotations
- Simpler parser implementation

## Performance Characteristics

### Time Complexity

- **Parsing**: O(n) where n is input size
- **Block lookup**: O(1) with map
- **List access**: O(1) indexed access

### Space Complexity

- **Memory**: O(n) for storing parsed structure
- **No backtracking**: No extra buffering needed
- **Streaming**: Can parse large files efficiently

### Optimizations

- Single-pass parsing
- No regular expressions in hot path
- Minimal allocations
- Reuse scanner buffer

## Testing Strategy

### Test Coverage

```
parser.go:      95% coverage
types.go:       100% coverage
Overall:        92% coverage
```

### Test Categories

1. **Unit Tests** - Individual functions
2. **Integration Tests** - Full parsing scenarios
3. **Edge Cases** - Empty input, malformed syntax
4. **Error Cases** - Invalid syntax, unclosed blocks

### Test Data

```
parser_test.go          # Main test file
examples/*.up           # Real-world examples
```

## CLI Tool Design

### Command Structure

```
up <command> [flags] [arguments]

Commands:
  parse     - Parse and display document
  validate  - Validate syntax
  convert   - Convert to other formats
```

### Implementation

Uses `github.com/urfave/cli/v2` for command-line interface:

- Consistent flag handling
- Auto-generated help
- Subcommand organization

## Future Enhancements

### Planned Features

- [ ] Schema validation support
- [ ] Namespace plugin system
- [ ] Template evaluation
- [ ] Streaming parser API
- [ ] Performance profiling

### Backward Compatibility

All changes maintain backward compatibility:
- Parser API remains stable
- Data structures are append-only
- New features use opt-in flags

## Design Decisions

### Why Interface{} for Value?

**Pros:**
- Supports heterogeneous collections
- Simple type switches
- No complex type hierarchies

**Cons:**
- No compile-time type safety
- Requires runtime type assertions

**Decision:** Flexibility outweighs type safety for a parser

### Why Strings for Scalars?

**Pros:**
- No conversion errors
- Preserves exact input
- Users control conversion

**Cons:**
- Users must handle conversion
- No built-in validation

**Decision:** Parser should preserve, not interpret

### Why Functional Options?

**Pros:**
- Configurable without breaking changes
- Self-documenting API
- Easy to extend

**Cons:**
- More verbose than setters
- Requires understanding pattern

**Decision:** Idiomatic Go, worth the verbosity

## Contributing

When contributing to the Go implementation:

1. **Follow Go conventions** - Use `gofmt`, `golint`
2. **Write tests** - Maintain >90% coverage
3. **Document public APIs** - Use godoc comments
4. **Keep it simple** - Prefer clarity over cleverness
5. **No dependencies** - Use only standard library

## References

- [UP Specification](https://github.com/uplang/spec)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)

