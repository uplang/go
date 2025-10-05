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
