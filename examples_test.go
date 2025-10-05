package up

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExampleFiles(t *testing.T) {
	examplesDir := "examples/core"

	// Test basic example files
	examples := []string{
		"01-basic-scalars.up",
		"02-blocks.up",
		"03-lists.up",
		"04-multiline.up",
		"06-comments.up",
	}

	for _, example := range examples {
		path := filepath.Join(examplesDir, example)
		t.Run(example, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("Example file not found: %s", path)
				return
			}

			parser := NewParser()
			doc, err := parser.ParseDocument(strings.NewReader(string(content)))
			if err != nil {
				t.Errorf("Failed to parse %s: %v", example, err)
				return
			}

			if doc == nil {
				t.Errorf("Parsed document is nil for %s", example)
				return
			}

			t.Logf("Successfully parsed %s with %d nodes", example, len(doc.Nodes))
		})
	}
}

