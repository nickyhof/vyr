package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nickyhof/vyr/internal/checker"
	"github.com/nickyhof/vyr/internal/compiler"
	"github.com/nickyhof/vyr/internal/lexer"
	"github.com/nickyhof/vyr/internal/loader"
	"github.com/nickyhof/vyr/internal/parser"
	"github.com/nickyhof/vyr/internal/vm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vyr <run|repl> [file.vyr]")
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: vyr run <file.vyr>")
		}
		return runFile(os.Args[2])
	case "repl":
		return runRepl()
	default:
		return fmt.Errorf("usage: vyr <run|repl> [file.vyr]")
	}
}

func runFile(filename string) error {
	searchPaths := stdlibSearchPaths()
	prog, err := loader.Load(filename, searchPaths...)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}

	if errs := checker.Check(prog); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "type error: %s\n", e.Message)
		}
		return fmt.Errorf("type checking failed with %d error(s)", len(errs))
	}

	code, err := compiler.New().Compile(prog)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	machine := vm.New(code)
	if err := machine.Run(); err != nil {
		return fmt.Errorf("runtime: %w", err)
	}

	return nil
}

func runRepl() error {
	fmt.Println("Vyr REPL v0.1 — type expressions or 'fn' declarations. Ctrl+D to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var fnDecls []string // accumulated function declarations

	for {
		fmt.Print("vyr> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Multi-line: read until braces balance
		input := line
		if strings.Contains(line, "{") {
			depth := strings.Count(input, "{") - strings.Count(input, "}")
			for depth > 0 {
				fmt.Print("...  ")
				if !scanner.Scan() {
					break
				}
				input += "\n" + scanner.Text()
				depth = strings.Count(input, "{") - strings.Count(input, "}")
			}
		}

		// If it's a function declaration, accumulate it
		if strings.HasPrefix(input, "fn ") && !strings.HasPrefix(input, "fn main") && !strings.HasPrefix(input, "fn(") {
			fnDecls = append(fnDecls, input)
			fmt.Println("  defined.")
			continue
		}

		// Wrap expression in main to execute
		source := strings.Join(fnDecls, "\n") + "\nfn main() {\n" + input + "\n|> print\n}"

		tokens := lexer.New(source).Tokenize()
		prog, err := parser.New(tokens).Parse()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  parse error: %v\n", err)
			continue
		}

		code, err := compiler.New().Compile(prog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  compile error: %v\n", err)
			continue
		}

		var out strings.Builder
		machine := vm.NewWithOutput(code, &out)
		if err := machine.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  runtime error: %v\n", err)
			continue
		}

		result := strings.TrimSpace(out.String())
		if result != "" && result != "nil" {
			fmt.Println(result)
		}
	}

	return nil
}

// stdlibSearchPaths returns directories to search for imports.
func stdlibSearchPaths() []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}
	return paths
}
