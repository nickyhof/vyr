package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nickyhof/vyr/internal/lexer"
	"github.com/nickyhof/vyr/internal/parser"
)

// Loader resolves imports and produces a merged AST Program.
type Loader struct {
	loaded      map[string]bool // absolute paths already loaded (cycle detection)
	searchPaths []string        // additional directories to search for imports
}

// Load reads a Vyr source file, resolves all imports recursively, and returns
// a merged Program with all declarations from every imported module.
// searchPaths provides additional directories to look for imports (e.g. stdlib).
func Load(filename string, searchPaths ...string) (*parser.Program, error) {
	l := &Loader{
		loaded:      make(map[string]bool),
		searchPaths: searchPaths,
	}
	return l.load(filename)
}

func (l *Loader) load(filename string) (*parser.Program, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolving path %s: %w", filename, err)
	}

	// Cycle detection
	if l.loaded[absPath] {
		return &parser.Program{}, nil // already loaded; skip
	}
	l.loaded[absPath] = true

	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}

	tokens := lexer.New(string(src)).Tokenize()
	prog, err := parser.New(tokens).Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filename, err)
	}

	dir := filepath.Dir(absPath)
	var merged []parser.Node

	for _, decl := range prog.Decls {
		imp, ok := decl.(*parser.ImportDecl)
		if !ok {
			merged = append(merged, decl)
			continue
		}

		importFile, err := l.resolve(imp.Path, dir)
		if err != nil {
			return nil, fmt.Errorf("import %q: %w", imp.Path, err)
		}

		imported, err := l.load(importFile)
		if err != nil {
			return nil, fmt.Errorf("import %q: %w", imp.Path, err)
		}

		// Merge non-import declarations from the imported file
		for _, d := range imported.Decls {
			if _, isImport := d.(*parser.ImportDecl); !isImport {
				merged = append(merged, d)
			}
		}
	}

	return &parser.Program{Decls: merged}, nil
}

// resolve finds an import path, trying relative to the importing file first,
// then each search path.
func (l *Loader) resolve(importPath, importingDir string) (string, error) {
	name := importPath + ".vyr"

	// 1. Relative to importing file
	candidate := filepath.Join(importingDir, name)
	if fileExists(candidate) {
		return candidate, nil
	}

	// 2. Search paths (e.g. project root for std/)
	for _, sp := range l.searchPaths {
		candidate = filepath.Join(sp, name)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("cannot find module %q (searched %s and %d search paths)", importPath, importingDir, len(l.searchPaths))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
