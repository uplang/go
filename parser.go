// Package up provides functional parsing capabilities for UP documents.
package up

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// UseDirective represents a !use directive with namespace list
type UseDirective struct {
	Namespaces []string
}

// Scanner wraps a bufio.Scanner with additional functionality.
type Scanner struct {
	*bufio.Scanner
	lineNum int
}

// NewScanner creates a new Scanner from an io.Reader.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		Scanner: bufio.NewScanner(r),
		lineNum: 0,
	}
}

// NextLine advances the scanner and returns the current line number and text.
func (s *Scanner) NextLine() (int, string, bool) {
	if !s.Scan() {
		return s.lineNum, "", false
	}
	s.lineNum++
	return s.lineNum, s.Text(), true
}

// ParseFunc represents a parsing function type.
type ParseFunc[T any] func(*Scanner, string) (T, error)

// Parser provides configurable parsing functionality.
type Parser struct {
	dedentFunc    func(string, int) string
	skipEmptyLine func(string) bool
	skipComment   func(string) bool
}

// NewParser creates a new Parser with default configuration.
func NewParser() *Parser {
	return &Parser{
		dedentFunc:    dedentLines,
		skipEmptyLine: func(line string) bool { return strings.TrimSpace(line) == "" },
		skipComment:   func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "#") },
	}
}

// WithDedentFunc configures the dedent function.
func (p *Parser) WithDedentFunc(fn func(string, int) string) *Parser {
	p.dedentFunc = fn
	return p
}

// WithSkipEmptyLine configures the empty line skip function.
func (p *Parser) WithSkipEmptyLine(fn func(string) bool) *Parser {
	p.skipEmptyLine = fn
	return p
}

// WithSkipComment configures the comment skip function.
func (p *Parser) WithSkipComment(fn func(string) bool) *Parser {
	p.skipComment = fn
	return p
}

// ParseDocument parses a UP document from an io.Reader.
func (p *Parser) ParseDocument(r io.Reader) (*Document, error) {
	scanner := NewScanner(r)
	nodes, err := p.parseNodes(scanner)
	if err != nil {
		return nil, err
	}

	return &Document{Nodes: nodes}, scanner.Err()
}

// parseNodes parses multiple nodes from the scanner.
func (p *Parser) parseNodes(scanner *Scanner) ([]Node, error) {
	var nodes []Node

	for {
		lineNum, line, ok := scanner.NextLine()
		if !ok {
			break
		}

		if p.skipEmptyLine(line) || p.skipComment(line) {
			continue
		}

		trimmedLine := strings.TrimSpace(line)

		// Handle document-level directives
		if strings.HasPrefix(trimmedLine, "!use") {
			useNode, err := p.parseUseDirective(scanner, trimmedLine)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			nodes = append(nodes, useNode)
			continue
		}

		if strings.HasPrefix(trimmedLine, "!lint") {
			lintNode, err := p.parseLintDirective(scanner, trimmedLine)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			nodes = append(nodes, lintNode)
			continue
		}

		node, err := p.parseLine(scanner, line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// parseUseDirective parses a !use directive: !use [namespace1, namespace2]
func (p *Parser) parseUseDirective(scanner *Scanner, line string) (Node, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "!use")
	line = strings.TrimSpace(line)

	// Parse the namespace list
	if strings.HasPrefix(line, "[") {
		namespaces, err := parseInlineList(line)
		if err != nil {
			return Node{}, fmt.Errorf("invalid !use directive: %w", err)
		}
		// Convert []any to []string
		nsList := make([]string, len(namespaces))
		for i, ns := range namespaces {
			if s, ok := ns.(string); ok {
				nsList[i] = s
			}
		}
		return Node{
			Key:   "_use",
			Type:  "directive",
			Value: UseDirective{Namespaces: nsList},
		}, nil
	}

	return Node{}, fmt.Errorf("!use directive requires a list: !use [namespace1, namespace2]")
}

// parseLintDirective parses a !lint directive block
func (p *Parser) parseLintDirective(scanner *Scanner, line string) (Node, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "!lint")
	line = strings.TrimSpace(line)

	// Expect a block: !lint { ... }
	if line == "{" {
		block, err := p.parseBlock(scanner)
		if err != nil {
			return Node{}, fmt.Errorf("invalid !lint block: %w", err)
		}
		return Node{
			Key:   "_lint",
			Type:  "directive",
			Value: block,
		}, nil
	}

	return Node{}, fmt.Errorf("!lint directive requires a block: !lint { ... }")
}

// parseLine parses a single key-value line.
func (p *Parser) parseLine(scanner *Scanner, line string) (Node, error) {
	keyPart, valPart, lineOriented := p.splitKeyValue(line)
	key, typeAnnotation := p.parseKeyAndType(keyPart)

	node := Node{
		Key:  key,
		Type: typeAnnotation,
	}

	// Handle !quoted annotation - preserves or adds literal quotes
	if typeAnnotation == "quoted" {
		// In line-oriented mode with !quoted, preserve/add quotes
		if !strings.HasPrefix(valPart, "\"") || !strings.HasSuffix(valPart, "\"") {
			valPart = "\"" + valPart + "\""
		}
		node.Type = "string" // Normalize type to string
		node.Value = valPart
		return node, nil
	}

	value, err := p.parseValue(scanner, node, valPart, lineOriented)
	if err != nil {
		return Node{}, err
	}

	node.Value = value
	return node, nil
}

// splitKeyValue splits a line into key and value parts.
// Supports both traditional whitespace-delimited and line-oriented (: suffix) syntax.
func (p *Parser) splitKeyValue(line string) (string, string, bool) {
	line = strings.TrimSpace(line)

	// Find where the key ends - either at whitespace or at end of line
	keyEnd := strings.IndexAny(line, " \t")
	if keyEnd == -1 {
		keyEnd = len(line)
	}

	keyPart := line[:keyEnd]

	// Check for line-oriented syntax: key ends with : (but not part of URL like https:)
	// The colon must be at the end of the key part (before whitespace)
	if strings.HasSuffix(keyPart, ":") && !strings.Contains(keyPart, "://") {
		key := strings.TrimSuffix(keyPart, ":")
		var value string
		if keyEnd < len(line) {
			value = strings.TrimSpace(line[keyEnd:])
			// Handle comments in line-oriented mode: # starts a comment
			if commentIdx := strings.Index(value, "#"); commentIdx >= 0 {
				value = strings.TrimSpace(value[:commentIdx])
			}
			// Strip surrounding quotes in line-oriented mode
			value = stripSurroundingQuotes(value)
		}
		return key, value, true // true = line-oriented mode
	}

	// Traditional whitespace-delimited syntax
	if keyEnd < len(line) {
		value := strings.TrimSpace(line[keyEnd:])
		// Handle quoted values in traditional mode
		value = stripSurroundingQuotes(value)
		return keyPart, value, false
	}

	return keyPart, "", false
}

// stripSurroundingQuotes removes surrounding double quotes from a value.
// "Hello World" -> Hello World
// "Quote" -> Quote
// Unquoted -> Unquoted (unchanged)
func stripSurroundingQuotes(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

// parseKeyAndType extracts key and type annotation from the key part.
func (p *Parser) parseKeyAndType(keyPart string) (string, string) {
	if idx := strings.Index(keyPart, "!"); idx >= 0 {
		return keyPart[:idx], keyPart[idx+1:]
	}
	return keyPart, ""
}

// parseValue parses the value part based on its format.
func (p *Parser) parseValue(scanner *Scanner, node Node, valPart string, lineOriented bool) (Value, error) {
	switch {
	case strings.HasPrefix(valPart, "```"):
		return p.parseMultiline(scanner, node, valPart)
	case valPart == "{":
		return p.parseBlock(scanner)
	case valPart == "[":
		return p.parseList(scanner)
	case strings.HasPrefix(valPart, "[") && strings.HasSuffix(valPart, "]"):
		// Inline list on same line: key [item1, item2, item3]
		return parseInlineList(valPart)
	case strings.HasPrefix(valPart, "{") && strings.Contains(valPart, "}"):
		// Inline block: key { ... } - parse as single-line block
		return p.parseInlineBlock(valPart)
	case node.Type == "table" && strings.HasPrefix(valPart, "{"):
		return p.parseTable(scanner)
	default:
		return valPart, nil
	}
}

// parseInlineBlock parses a single-line block: { key1 value1, key2 value2 }
func (p *Parser) parseInlineBlock(s string) (Block, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	s = strings.TrimSpace(s)

	if s == "" {
		return make(Block), nil
	}

	block := make(Block)
	// Split by comma for inline block syntax
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Each part is "key value" or "key!type value"
		keyPart, valPart, _ := p.splitKeyValue(part)
		key, _ := p.parseKeyAndType(keyPart)
		block[key] = valPart
	}
	return block, nil
}

// parseMultiline handles triple-backtick blocks with optional dedent.
func (p *Parser) parseMultiline(scanner *Scanner, node Node, line string) (string, error) {
	_ = strings.TrimSpace(strings.TrimPrefix(line, "```")) // lang hint not used in current implementation
	var content []string

	for {
		_, line, ok := scanner.NextLine()
		if !ok {
			break
		}
		if strings.TrimSpace(line) == "```" {
			break
		}
		content = append(content, line)
	}

	text := strings.Join(content, "\n")

	if node.Type != "" {
		if dedent, err := strconv.Atoi(node.Type); err == nil {
			text = p.dedentFunc(text, dedent)
		}
	}

	return text, nil
}

// parseBlock parses a standard { ... } block of statements.
func (p *Parser) parseBlock(scanner *Scanner) (Block, error) {
	block := make(Block)

	for {
		_, line, ok := scanner.NextLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if line == "}" {
			break
		}
		if p.skipEmptyLine(line) || p.skipComment(line) {
			continue
		}

		node, err := p.parseLine(scanner, line)
		if err != nil {
			return nil, err
		}
		block[node.Key] = node.Value
	}

	return block, nil
}

// parseList parses a [...] list.
func (p *Parser) parseList(scanner *Scanner) (List, error) {
	var list List

	for {
		_, line, ok := scanner.NextLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if line == "]" {
			break
		}
		if p.skipEmptyLine(line) || p.skipComment(line) {
			continue
		}

		item, err := p.parseListItem(scanner, line)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}

// parseListItem parses a single list item.
func (p *Parser) parseListItem(scanner *Scanner, line string) (Value, error) {
	switch {
	case strings.HasPrefix(line, "{"):
		return p.parseBlock(scanner)
	case strings.HasPrefix(line, "["):
		return parseInlineList(line)
	default:
		return line, nil
	}
}

// parseTable parses a table: columns + rows.
func (p *Parser) parseTable(scanner *Scanner) (map[string]any, error) {
	table := make(map[string]any)

	for {
		_, line, ok := scanner.NextLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if line == "}" {
			break
		}
		if p.skipEmptyLine(line) || p.skipComment(line) {
			continue
		}

		if strings.HasPrefix(line, "columns") {
			colList, err := parseInlineList(line[len("columns"):])
			if err != nil {
				return nil, err
			}
			table["columns"] = colList
		} else if strings.HasPrefix(line, "rows") {
			rowsBlock, err := p.parseBlockOfLists(scanner)
			if err != nil {
				return nil, err
			}
			table["rows"] = rowsBlock
		}
	}

	return table, nil
}

// parseBlockOfLists parses multiple [...] rows inside rows { ... }.
func (p *Parser) parseBlockOfLists(scanner *Scanner) ([]any, error) {
	var rows []any

	for {
		_, line, ok := scanner.NextLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if line == "}" {
			break
		}
		if p.skipEmptyLine(line) || p.skipComment(line) {
			continue
		}
		if strings.HasPrefix(line, "[") {
			list, err := parseInlineList(line)
			if err != nil {
				return nil, err
			}
			rows = append(rows, list)
		}
	}

	return rows, nil
}

// parseInlineList parses a single inline list: [item1, item2, ...].
func parseInlineList(line string) ([]any, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "[")
	line = strings.TrimSuffix(line, "]")

	if line == "" {
		return []any{}, nil
	}

	items := strings.Split(line, ",")
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = strings.TrimSpace(item)
	}
	return result, nil
}

// dedentLines removes N spaces from the beginning of each line.
func dedentLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) >= n {
			lines[i] = line[n:]
		}
	}
	return strings.Join(lines, "\n")
}
