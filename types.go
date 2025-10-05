// Package up defines the core data structures for UP parsing.
package up

// Value represents any UP value.
type Value any

// Node represents a key-value pair with optional type annotation.
type Node struct {
	Key   string // The key name
	Type  string // Optional type annotation (e.g., "int", "bool", "string")
	Value Value  // The parsed value (string, Block, List, Table, or UseDirective)
}

// Document represents a parsed UP document.
type Document struct {
	Nodes []Node // Ordered list of top-level nodes
}

// Block represents a UP block structure { ... }.
type Block map[string]Value

// List represents a UP list structure [ ... ].
type List []Value

// Table represents a UP table with columns and rows.
type Table struct {
	Columns []any
	Rows    []any
}

// LintRule represents a lint rule with its enforcement level.
type LintRule struct {
	Name  string // Rule name (e.g., "no-empty-values")
	Level string // Enforcement level: "error", "warning", "info"
}
