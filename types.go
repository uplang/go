// Package up defines the core data structures for UP parsing.
package up

// Value represents any UP value.
type Value any

// Node represents a key-value pair with optional type annotation.
type Node struct {
	Key   string
	Type  string
	Value Value
}

// Document represents a parsed UP document.
type Document struct {
	Nodes []Node
}

// Block represents a UP block structure.
type Block map[string]Value

// List represents a UP list structure.
type List []Value

// Table represents a UP table with columns and rows.
type Table struct {
	Columns []any
	Rows    []any
}
