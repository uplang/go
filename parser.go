// Package up provides functional parsing capabilities for UP documents.
package up

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

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

		node, err := p.parseLine(scanner, line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// parseLine parses a single key-value line.
func (p *Parser) parseLine(scanner *Scanner, line string) (Node, error) {
	keyPart, valPart := p.splitKeyValue(line)
	key, typeAnnotation := p.parseKeyAndType(keyPart)

	node := Node{
		Key:  key,
		Type: typeAnnotation,
	}

	value, err := p.parseValue(scanner, node, valPart)
	if err != nil {
		return Node{}, err
	}

	node.Value = value
	return node, nil
}

// splitKeyValue splits a line into key and value parts.
func (p *Parser) splitKeyValue(line string) (string, string) {
	if idx := strings.IndexAny(line, " \t"); idx >= 0 {
		return line[:idx], strings.TrimSpace(line[idx:])
	}
	return line, ""
}

// parseKeyAndType extracts key and type annotation from the key part.
func (p *Parser) parseKeyAndType(keyPart string) (string, string) {
	if idx := strings.Index(keyPart, "!"); idx >= 0 {
		return keyPart[:idx], keyPart[idx+1:]
	}
	return keyPart, ""
}

// parseValue parses the value part based on its format.
func (p *Parser) parseValue(scanner *Scanner, node Node, valPart string) (Value, error) {
	switch {
	case strings.HasPrefix(valPart, "```"):
		return p.parseMultiline(scanner, node, valPart)
	case valPart == "{":
		return p.parseBlock(scanner)
	case valPart == "[":
		return p.parseList(scanner)
	case node.Type == "table" && strings.HasPrefix(valPart, "{"):
		return p.parseTable(scanner)
	default:
		return valPart, nil
	}
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
