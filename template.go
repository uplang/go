// Package up provides templating support for UP documents
package up

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TemplateEngine processes UP templates with overlays, includes, and variables
type TemplateEngine struct {
	options TemplateOptions
	vars    map[string]any
	visited map[string]bool // prevent circular dependencies
}

// TemplateOptions configures template processing
type TemplateOptions struct {
	MergeStrategy string // "deep", "shallow", "replace"
	ListStrategy  string // "append", "replace", "unique"
	BaseDir       string // base directory for relative includes
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		options: TemplateOptions{
			MergeStrategy: "deep",
			ListStrategy:  "append",
			BaseDir:       ".",
		},
		vars:    make(map[string]any),
		visited: make(map[string]bool),
	}
}

// WithOptions sets template options
func (e *TemplateEngine) WithOptions(opts TemplateOptions) *TemplateEngine {
	e.options = opts
	return e
}

// WithVars sets initial variables
func (e *TemplateEngine) WithVars(vars map[string]any) *TemplateEngine {
	e.vars = vars
	return e
}

// ProcessTemplate processes a UP template file
func (e *TemplateEngine) ProcessTemplate(filename string) (*Document, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check for circular dependencies
	if e.visited[absPath] {
		return nil, fmt.Errorf("circular dependency detected: %s", filename)
	}
	e.visited[absPath] = true
	defer delete(e.visited, absPath)

	// Parse the file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.ParseDocument(file)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Update base directory for relative includes
	dir := filepath.Dir(absPath)
	oldBaseDir := e.options.BaseDir
	e.options.BaseDir = dir
	defer func() { e.options.BaseDir = oldBaseDir }()

	// Process template directives
	return e.processDocument(doc)
}

// processDocument processes template directives in a document
func (e *TemplateEngine) processDocument(doc *Document) (*Document, error) {
	result := &Document{Nodes: []Node{}}
	var baseDoc *Document
	var overlayNodes []Node
	var patchNodes []Node
	var includeFiles []string
	var allDocs []*Document // Collect all documents for variable extraction

	// Extract template directives based on type annotations
	for _, node := range doc.Nodes {
		// Check for template annotations (!base, !overlay, !include, !patch, !merge)
		switch node.Type {
		case "base":
			// Load base file (don't process yet, just parse)
			if baseFile, ok := node.Value.(string); ok {
				basePath := filepath.Join(e.options.BaseDir, baseFile)
				var err error
				baseDoc, err = e.loadDocumentRaw(basePath)
				if err != nil {
					return nil, fmt.Errorf("failed to load base %s: %w", baseFile, err)
				}
				allDocs = append(allDocs, baseDoc)
			}
		case "overlay":
			// Store overlay nodes - the key is the block name, value is what to merge
			if block, ok := node.Value.(Block); ok {
				// This is a block to overlay
				overlayNodes = append(overlayNodes, Node{Key: node.Key, Value: block, Type: node.Type})
			}
		case "include":
			// Store include files
			if list, ok := node.Value.(List); ok {
				for _, item := range list {
					if file, ok := item.(string); ok {
						includeFiles = append(includeFiles, file)
					}
				}
			}
		case "patch":
			// Store patch directives
			if block, ok := node.Value.(Block); ok {
				for k, v := range block {
					patchNodes = append(patchNodes, Node{Key: k, Value: v})
				}
			}
		case "merge":
			// Update merge options
			if block, ok := node.Value.(Block); ok {
				if strategy, ok := block["strategy"].(string); ok {
					e.options.MergeStrategy = strategy
				}
				if listStrategy, ok := block["list_strategy"].(string); ok {
					e.options.ListStrategy = listStrategy
				}
			}
		default:
			// Store all non-template nodes
			result.Nodes = append(result.Nodes, node)
		}
	}

	// Load all included files
	for _, includeFile := range includeFiles {
		includePath := filepath.Join(e.options.BaseDir, includeFile)
		includeDoc, err := e.loadDocumentRaw(includePath)
		if err != nil {
			return nil, fmt.Errorf("failed to include %s: %w", includeFile, err)
		}
		allDocs = append(allDocs, includeDoc)
	}

	// Add current document to the list
	allDocs = append(allDocs, result)

	// Extract all variables from all documents BEFORE merging
	// This allows variables to reference each other regardless of order
	for _, d := range allDocs {
		for _, node := range d.Nodes {
			if node.Key == "vars" {
				if block, ok := node.Value.(Block); ok {
					e.extractVars(block, "")
				}
			}
		}
	}

	// Build final document by merging all documents
	finalDoc := &Document{Nodes: []Node{}}

	// 1. Start with base if present
	if baseDoc != nil {
		finalDoc = baseDoc
	}

	// 2. Merge all included documents
	for _, includeDoc := range allDocs[len(allDocs)-len(includeFiles)-1 : len(allDocs)-1] {
		if includeDoc != result { // Skip the current doc, we'll merge it next
			finalDoc = e.mergeDocuments(finalDoc, includeDoc)
		}
	}

	// 3. Merge current document nodes
	if len(result.Nodes) > 0 {
		finalDoc = e.mergeDocuments(finalDoc, result)
	}

	// 4. Apply overlays - merge each overlay block with corresponding block in final doc
	for _, overlayNode := range overlayNodes {
		// Find the target node in finalDoc and merge
		merged := false
		for i, targetNode := range finalDoc.Nodes {
			if targetNode.Key == overlayNode.Key {
				// Merge the overlay value with the target value
				finalDoc.Nodes[i].Value = e.mergeValues(targetNode.Value, overlayNode.Value)
				merged = true
				break
			}
		}
		// If not found, add as new node
		if !merged {
			finalDoc.Nodes = append(finalDoc.Nodes, Node{Key: overlayNode.Key, Value: overlayNode.Value})
		}
	}

	// 5. Apply patches
	if len(patchNodes) > 0 {
		finalDoc = e.applyPatches(finalDoc, patchNodes)
	}

	// 6. Iteratively resolve variable references until convergence or circular dependency
	finalDoc, err := e.resolveVariablesIteratively(finalDoc)
	if err != nil {
		return nil, err
	}

	return finalDoc, nil
}

// loadDocumentRaw loads and parses a document without processing template directives
func (e *TemplateEngine) loadDocumentRaw(filename string) (*Document, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check for circular dependencies
	if e.visited[absPath] {
		return nil, fmt.Errorf("circular dependency detected: %s", filename)
	}
	e.visited[absPath] = true
	defer delete(e.visited, absPath)

	// Parse the file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.ParseDocument(file)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Update base directory for relative includes
	oldBaseDir := e.options.BaseDir
	e.options.BaseDir = filepath.Dir(absPath)
	defer func() { e.options.BaseDir = oldBaseDir }()

	// Recursively process this document
	return e.processDocument(doc)
}

// extractVars extracts variables from a block
// Variables can contain references to other variables, which will be resolved iteratively
func (e *TemplateEngine) extractVars(block Block, prefix string) {
	for k, v := range block {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		if nestedBlock, ok := v.(Block); ok {
			e.extractVars(nestedBlock, path)
		} else {
			// Store the value as-is; it will be resolved iteratively
			// This allows variables to reference other variables
			e.vars[path] = v
		}
	}
}

// resolveVariablesIteratively resolves variable references iteratively until convergence
func (e *TemplateEngine) resolveVariablesIteratively(doc *Document) (*Document, error) {
	const maxIterations = 100 // Prevent infinite loops

	// First, iteratively resolve the variables map itself
	// This allows variables to reference other variables
	for iteration := 0; iteration < maxIterations; iteration++ {
		hasChanges := false
		newVars := make(map[string]any)

		// Try to resolve each variable
		for path, value := range e.vars {
			resolved := e.resolveValue(value)
			newVars[path] = resolved

			// Check if this variable changed
			if !valuesEqual(value, resolved) {
				hasChanges = true
			}
		}

		// Update the vars map with resolved values
		e.vars = newVars

		// If nothing changed, variables are fully resolved
		if !hasChanges {
			break
		}

		// Check if we hit max iterations (circular dependency)
		if iteration == maxIterations-1 {
			return nil, fmt.Errorf("circular dependency detected: exceeded %d iterations resolving variables", maxIterations)
		}
	}

	// Now resolve all variable references in the document using the fully resolved vars
	result := &Document{Nodes: make([]Node, len(doc.Nodes))}
	for i, node := range doc.Nodes {
		result.Nodes[i] = Node{
			Key:  node.Key,
			Type: node.Type,
			Value: e.resolveValue(node.Value),
		}
	}

	return result, nil
}

// valuesEqual checks if two values are equal (for detecting convergence)
func valuesEqual(a, b any) bool {
	switch va := a.(type) {
	case string:
		vb, ok := b.(string)
		return ok && va == vb
	case Block:
		vb, ok := b.(Block)
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !valuesEqual(v, vb[k]) {
				return false
			}
		}
		return true
	case List:
		vb, ok := b.(List)
		if !ok || len(va) != len(vb) {
			return false
		}
		for i, v := range va {
			if !valuesEqual(v, vb[i]) {
				return false
			}
		}
		return true
	case int, int64, float64, bool:
		return a == b
	default:
		return false
	}
}

// resolveValue resolves $vars references in a value
func (e *TemplateEngine) resolveValue(value any) any {
	switch v := value.(type) {
	case string:
		// Handle strings that may contain one or more $vars. references
		if !strings.Contains(v, "$vars.") {
			return v // No variable references, return as-is
		}

		// Replace all $vars. references in the string
		result := v
		// Find all $vars.path patterns and replace them
		for strings.Contains(result, "$vars.") {
			oldResult := result

			// Find the start of a variable reference
			start := strings.Index(result, "$vars.")
			if start == -1 {
				break
			}

			// Find the end of the variable path (alphanumeric, underscore, dot)
			end := start + 6 // len("$vars.")
			for end < len(result) && (isVarChar(result[end]) || result[end] == '.') {
				end++
			}

			// Extract the variable path
			fullRef := result[start:end]
			varPath := strings.TrimPrefix(fullRef, "$vars.")

			// Look up the variable
			if resolved, ok := e.vars[varPath]; ok {
				// Convert resolved value to string
				resolvedStr := fmt.Sprint(resolved)

				// If the entire string is just this one variable reference, return the actual type
				if result == fullRef {
					return e.resolveValue(resolved)
				}

				// Otherwise, do string replacement
				result = strings.Replace(result, fullRef, resolvedStr, 1)
			} else {
				// Variable not found, skip this one to avoid infinite loop
				// Replace with a marker temporarily
				result = strings.Replace(result, fullRef, "<<<"+fullRef+">>>", 1)
			}

			// If nothing changed, break to avoid infinite loop
			if result == oldResult {
				break
			}
		}

		// Restore markers for unresolved variables
		result = strings.ReplaceAll(result, "<<<$vars.", "$vars.")
		result = strings.ReplaceAll(result, ">>>", "")

		return result
	case Block:
		result := make(Block)
		for k, val := range v {
			result[k] = e.resolveValue(val)
		}
		return result
	case List:
		result := make(List, len(v))
		for i, val := range v {
			result[i] = e.resolveValue(val)
		}
		return result
	default:
		return v
	}
}

// isVarChar checks if a character is valid in a variable path
func isVarChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// mergeDocuments merges two documents according to merge strategy
func (e *TemplateEngine) mergeDocuments(base, overlay *Document) *Document {
	result := &Document{Nodes: make([]Node, 0)}

	// Create a map of base nodes
	baseMap := make(map[string]Node)
	for _, node := range base.Nodes {
		baseMap[node.Key] = node
	}

	// Merge overlay nodes
	for _, overlayNode := range overlay.Nodes {
		if baseNode, exists := baseMap[overlayNode.Key]; exists {
			// Merge values
			merged := e.mergeValues(baseNode.Value, overlayNode.Value)
			result.Nodes = append(result.Nodes, Node{
				Key:  overlayNode.Key,
				Type: overlayNode.Type,
				Value: merged,
			})
			delete(baseMap, overlayNode.Key)
		} else {
			// Add new node
			result.Nodes = append(result.Nodes, overlayNode)
		}
	}

	// Add remaining base nodes
	for _, node := range base.Nodes {
		if _, merged := baseMap[node.Key]; merged {
			result.Nodes = append(result.Nodes, node)
		}
	}

	return result
}

// mergeValues merges two values according to merge strategy
func (e *TemplateEngine) mergeValues(base, overlay any) any {
	// If overlay is nil, keep base
	if overlay == nil {
		return base
	}

	// Type mismatch or shallow merge: replace
	if e.options.MergeStrategy == "shallow" || e.options.MergeStrategy == "replace" {
		return overlay
	}

	// Deep merge for blocks
	baseBlock, baseIsBlock := base.(Block)
	overlayBlock, overlayIsBlock := overlay.(Block)
	if baseIsBlock && overlayIsBlock && e.options.MergeStrategy == "deep" {
		result := make(Block)
		// Copy base
		for k, v := range baseBlock {
			result[k] = v
		}
		// Merge overlay
		for k, v := range overlayBlock {
			if existing, exists := result[k]; exists {
				result[k] = e.mergeValues(existing, v)
			} else {
				result[k] = v
			}
		}
		return result
	}

	// List merge strategies
	baseList, baseIsList := base.(List)
	overlayList, overlayIsList := overlay.(List)
	if baseIsList && overlayIsList {
		switch e.options.ListStrategy {
		case "append":
			return append(baseList, overlayList...)
		case "unique":
			return e.uniqueList(append(baseList, overlayList...))
		case "replace":
			return overlayList
		default:
			return overlayList
		}
	}

	// Default: replace
	return overlay
}

// uniqueList returns a list with unique string values
func (e *TemplateEngine) uniqueList(list List) List {
	seen := make(map[string]bool)
	result := List{}
	for _, item := range list {
		if str, ok := item.(string); ok {
			if !seen[str] {
				seen[str] = true
				result = append(result, item)
			}
		} else {
			result = append(result, item)
		}
	}
	return result
}

// applyPatches applies patch directives to a document
func (e *TemplateEngine) applyPatches(doc *Document, patches []Node) *Document {
	result := &Document{Nodes: make([]Node, len(doc.Nodes))}
	copy(result.Nodes, doc.Nodes)

	for _, patch := range patches {
		// Parse patch path (e.g., "server.host", "servers[*].cpu")
		parts := strings.Split(patch.Key, ".")
		e.applyPatchPath(result, parts, patch.Value)
	}

	return result
}

// applyPatchPath applies a patch at a specific path
func (e *TemplateEngine) applyPatchPath(doc *Document, path []string, value any) {
	if len(path) == 0 {
		return
	}

	// Find the target node
	for i, node := range doc.Nodes {
		if node.Key == path[0] {
			if len(path) == 1 {
				// Direct replacement
				doc.Nodes[i].Value = value
			} else {
				// Navigate deeper
				if block, ok := node.Value.(Block); ok {
					e.applyPatchToBlock(block, path[1:], value)
				}
			}
			return
		}
	}
}

// applyPatchToBlock applies a patch within a block
func (e *TemplateEngine) applyPatchToBlock(block Block, path []string, value any) {
	if len(path) == 0 {
		return
	}

	key := path[0]

	// Handle list indexing: key[*], key[0], key[name=value]
	if strings.Contains(key, "[") {
		// Extract base key and selector
		parts := strings.SplitN(key, "[", 2)
		baseKey := parts[0]
		selector := strings.TrimSuffix(parts[1], "]")

		if list, ok := block[baseKey].(List); ok {
			if selector == "*" {
				// Apply to all items
				for i := range list {
					if len(path) == 1 {
						list[i] = value
					} else if itemBlock, ok := list[i].(Block); ok {
						e.applyPatchToBlock(itemBlock, path[1:], value)
					}
				}
			}
			// Could add numeric index and key=value selectors here
		}
		return
	}

	if len(path) == 1 {
		// Direct set
		block[key] = value
	} else {
		// Navigate deeper
		if nestedBlock, ok := block[key].(Block); ok {
			e.applyPatchToBlock(nestedBlock, path[1:], value)
		}
	}
}

// ProcessTemplateFromReader processes a template from an io.Reader
func (e *TemplateEngine) ProcessTemplateFromReader(r io.Reader) (*Document, error) {
	parser := NewParser()
	doc, err := parser.ParseDocument(r)
	if err != nil {
		return nil, err
	}
	return e.processDocument(doc)
}

