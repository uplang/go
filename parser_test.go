package up

import (
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatal("NewParser() returned nil")
	}
}

func TestParseDocument_Simple(t *testing.T) {
	input := `name John Doe
age!int 30
active!bool true`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(doc.Nodes))
	}

	// Check first node
	if doc.Nodes[0].Key != "name" {
		t.Errorf("Expected key 'name', got '%s'", doc.Nodes[0].Key)
	}
	if doc.Nodes[0].Value != "John Doe" {
		t.Errorf("Expected value 'John Doe', got '%s'", doc.Nodes[0].Value)
	}

	// Check second node with type annotation
	if doc.Nodes[1].Key != "age" {
		t.Errorf("Expected key 'age', got '%s'", doc.Nodes[1].Key)
	}
	if doc.Nodes[1].Type != "int" {
		t.Errorf("Expected type 'int', got '%s'", doc.Nodes[1].Type)
	}
	if doc.Nodes[1].Value != "30" {
		t.Errorf("Expected value '30', got '%s'", doc.Nodes[1].Value)
	}
}

func TestParseDocument_Multiline(t *testing.T) {
	input := "description!4 ```markdown\n    # About this service\n    - Written by Robert\n    - Runs on Postgres\n```\n"

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	node := doc.Nodes[0]
	if node.Key != "description" {
		t.Errorf("Expected key 'description', got '%s'", node.Key)
	}

	expectedContent := "# About this service\n- Written by Robert\n- Runs on Postgres"
	if node.Value != expectedContent {
		t.Errorf("Expected dedented content, got: %s", node.Value)
	}
}

func TestParseDocument_Block(t *testing.T) {
	input := `server {
host localhost
port!int 8080
}`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	node := doc.Nodes[0]
	if node.Key != "server" {
		t.Errorf("Expected key 'server', got '%s'", node.Key)
	}

	block, ok := node.Value.(Block)
	if !ok {
		t.Fatalf("Expected Block type, got %T", node.Value)
	}

	if block["host"] != "localhost" {
		t.Errorf("Expected host 'localhost', got '%v'", block["host"])
	}

	if block["port"] != "8080" {
		t.Errorf("Expected port '8080', got '%v'", block["port"])
	}
}

func TestParseDocument_List(t *testing.T) {
	input := `items [
apple
banana
cherry
]`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	node := doc.Nodes[0]
	if node.Key != "items" {
		t.Errorf("Expected key 'items', got '%s'", node.Key)
	}

	list, ok := node.Value.(List)
	if !ok {
		t.Fatalf("Expected List type, got %T", node.Value)
	}

	expectedItems := []string{"apple", "banana", "cherry"}
	if len(list) != len(expectedItems) {
		t.Fatalf("Expected %d items, got %d", len(expectedItems), len(list))
	}

	for i, expected := range expectedItems {
		if list[i] != expected {
			t.Errorf("Expected item[%d] '%s', got '%v'", i, expected, list[i])
		}
	}
}

func TestParseDocument_EmptyLinesAndComments(t *testing.T) {
	input := `# This is a comment
name John

# Another comment
age!int 30

`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].Key != "name" {
		t.Errorf("Expected key 'name', got '%s'", doc.Nodes[0].Key)
	}

	if doc.Nodes[1].Key != "age" {
		t.Errorf("Expected key 'age', got '%s'", doc.Nodes[1].Key)
	}
}

func TestParseInlineList(t *testing.T) {
	tests := []struct {
		input    string
		expected []any
	}{
		{"[apple, banana, cherry]", []any{"apple", "banana", "cherry"}},
		{"[]", []any{}},
		{"[single]", []any{"single"}},
		{"[item1,item2,item3]", []any{"item1", "item2", "item3"}},
	}

	for _, test := range tests {
		result, err := parseInlineList(test.input)
		if err != nil {
			t.Errorf("parseInlineList(%s) failed: %v", test.input, err)
			continue
		}

		if len(result) != len(test.expected) {
			t.Errorf("parseInlineList(%s) length mismatch: expected %d, got %d",
				test.input, len(test.expected), len(result))
			continue
		}

		for i, expected := range test.expected {
			if result[i] != expected {
				t.Errorf("parseInlineList(%s)[%d]: expected %v, got %v",
					test.input, i, expected, result[i])
			}
		}
	}
}

func TestDedentLines(t *testing.T) {
	input := "    line1\n    line2\n    line3"
	expected := "line1\nline2\nline3"

	result := dedentLines(input, 4)
	if result != expected {
		t.Errorf("dedentLines() failed:\nexpected: %q\ngot: %q", expected, result)
	}
}

func TestParserWithCustomDedent(t *testing.T) {
	customDedent := func(s string, n int) string {
		return strings.ReplaceAll(s, "\t", strings.Repeat(" ", n))
	}

	p := NewParser().WithDedentFunc(customDedent)

	input := "content!2 ```\n\tindented content\n\tmore content\n```\n"

	doc, err := p.ParseDocument(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	// The custom dedent function should replace tabs with 2 spaces
	expected := "  indented content\n  more content"
	if doc.Nodes[0].Value != expected {
		t.Errorf("Expected custom dedent result, got: %q", doc.Nodes[0].Value)
	}
}

// New tests for spec-compliant features

func TestParseDocument_LineOriented(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		key      string
		expected string
	}{
		{
			name:     "simple line-oriented",
			input:    "name: John Doe",
			key:      "name",
			expected: "John Doe",
		},
		{
			name:     "line-oriented with extra spaces",
			input:    "description: This is a complete sentence.",
			key:      "description",
			expected: "This is a complete sentence.",
		},
		{
			name:     "line-oriented strips surrounding quotes",
			input:    `title: "Senior Engineer"`,
			key:      "title",
			expected: "Senior Engineer",
		},
		{
			name:     "line-oriented with comment",
			input:    "message: Hello World  # this is a comment",
			key:      "message",
			expected: "Hello World",
		},
		{
			name:     "URL value (colon not at end)",
			input:    "website https://example.com",
			key:      "website",
			expected: "https://example.com",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := p.ParseDocument(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ParseDocument() failed: %v", err)
			}

			if len(doc.Nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
			}

			if doc.Nodes[0].Key != tt.key {
				t.Errorf("Expected key '%s', got '%s'", tt.key, doc.Nodes[0].Key)
			}

			if doc.Nodes[0].Value != tt.expected {
				t.Errorf("Expected value '%s', got '%s'", tt.expected, doc.Nodes[0].Value)
			}
		})
	}
}

func TestParseDocument_LineOrientedWithType(t *testing.T) {
	input := `name!string: John Doe
count!int: 42
enabled!bool: true`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(doc.Nodes))
	}

	// Check first node
	if doc.Nodes[0].Key != "name" || doc.Nodes[0].Type != "string" || doc.Nodes[0].Value != "John Doe" {
		t.Errorf("First node mismatch: %+v", doc.Nodes[0])
	}

	// Check second node
	if doc.Nodes[1].Key != "count" || doc.Nodes[1].Type != "int" || doc.Nodes[1].Value != "42" {
		t.Errorf("Second node mismatch: %+v", doc.Nodes[1])
	}

	// Check third node
	if doc.Nodes[2].Key != "enabled" || doc.Nodes[2].Type != "bool" || doc.Nodes[2].Value != "true" {
		t.Errorf("Third node mismatch: %+v", doc.Nodes[2])
	}
}

func TestParseDocument_QuotedAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "quoted preserves quotes",
			input:    `title!quoted: "Senior Engineer"`,
			expected: `"Senior Engineer"`,
		},
		{
			name:     "quoted adds quotes",
			input:    `name!quoted: Alice`,
			expected: `"Alice"`,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := p.ParseDocument(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ParseDocument() failed: %v", err)
			}

			if len(doc.Nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
			}

			if doc.Nodes[0].Value != tt.expected {
				t.Errorf("Expected value '%s', got '%s'", tt.expected, doc.Nodes[0].Value)
			}

			// Type should be normalized to "string"
			if doc.Nodes[0].Type != "string" {
				t.Errorf("Expected type 'string', got '%s'", doc.Nodes[0].Type)
			}
		})
	}
}

func TestParseDocument_InlineList(t *testing.T) {
	input := "colors [red, green, blue]"

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	list, ok := doc.Nodes[0].Value.([]any)
	if !ok {
		t.Fatalf("Expected []any type, got %T", doc.Nodes[0].Value)
	}

	expected := []string{"red", "green", "blue"}
	if len(list) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(list))
	}

	for i, exp := range expected {
		if list[i] != exp {
			t.Errorf("Expected item[%d] '%s', got '%v'", i, exp, list[i])
		}
	}
}

func TestParseDocument_UseDirective(t *testing.T) {
	input := "!use [time, id, faker, random]"

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].Key != "_use" {
		t.Errorf("Expected key '_use', got '%s'", doc.Nodes[0].Key)
	}

	if doc.Nodes[0].Type != "directive" {
		t.Errorf("Expected type 'directive', got '%s'", doc.Nodes[0].Type)
	}

	useDir, ok := doc.Nodes[0].Value.(UseDirective)
	if !ok {
		t.Fatalf("Expected UseDirective type, got %T", doc.Nodes[0].Value)
	}

	expected := []string{"time", "id", "faker", "random"}
	if len(useDir.Namespaces) != len(expected) {
		t.Fatalf("Expected %d namespaces, got %d", len(expected), len(useDir.Namespaces))
	}

	for i, exp := range expected {
		if useDir.Namespaces[i] != exp {
			t.Errorf("Expected namespace[%d] '%s', got '%s'", i, exp, useDir.Namespaces[i])
		}
	}
}

func TestParseDocument_LintDirective(t *testing.T) {
	input := `!lint {
  no-empty-values!level warning
  require-type-annotations!level error
}`

	p := NewParser()
	doc, err := p.ParseDocument(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDocument() failed: %v", err)
	}

	if len(doc.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].Key != "_lint" {
		t.Errorf("Expected key '_lint', got '%s'", doc.Nodes[0].Key)
	}

	if doc.Nodes[0].Type != "directive" {
		t.Errorf("Expected type 'directive', got '%s'", doc.Nodes[0].Type)
	}

	block, ok := doc.Nodes[0].Value.(Block)
	if !ok {
		t.Fatalf("Expected Block type, got %T", doc.Nodes[0].Value)
	}

	if block["no-empty-values"] != "warning" {
		t.Errorf("Expected no-empty-values warning, got %v", block["no-empty-values"])
	}

	if block["require-type-annotations"] != "error" {
		t.Errorf("Expected require-type-annotations error, got %v", block["require-type-annotations"])
	}
}

func TestParseDocument_TraditionalQuotedValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		key      string
		expected string
	}{
		{
			name:     "traditional quoted value",
			input:    `name "John Doe"`,
			key:      "name",
			expected: "John Doe",
		},
		{
			name:     "single word no quotes",
			input:    "status active",
			key:      "status",
			expected: "active",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := p.ParseDocument(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ParseDocument() failed: %v", err)
			}

			if len(doc.Nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(doc.Nodes))
			}

			if doc.Nodes[0].Key != tt.key {
				t.Errorf("Expected key '%s', got '%s'", tt.key, doc.Nodes[0].Key)
			}

			if doc.Nodes[0].Value != tt.expected {
				t.Errorf("Expected value '%s', got '%s'", tt.expected, doc.Nodes[0].Value)
			}
		})
	}
}
