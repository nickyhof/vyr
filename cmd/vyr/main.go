package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickyhof/vyr/internal/checker"
	"github.com/nickyhof/vyr/internal/codegen"
	"github.com/nickyhof/vyr/internal/loader"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vyr <run|build> <file.vyr>")
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: vyr run <file.vyr>")
		}
		return runFile(os.Args[2])
	case "build":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: vyr build <file.vyr> [-o output]")
		}
		output := ""
		if len(os.Args) >= 5 && os.Args[3] == "-o" {
			output = os.Args[4]
		}
		return buildFile(os.Args[2], output)
	default:
		return fmt.Errorf("usage: vyr <run|build> <file.vyr>")
	}
}

func compile(filename string) (string, error) {
	searchPaths := stdlibSearchPaths()
	prog, err := loader.Load(filename, searchPaths...)
	if err != nil {
		return "", fmt.Errorf("load: %w", err)
	}

	if errs := checker.Check(prog); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "type error: %s\n", e.Message)
		}
		return "", fmt.Errorf("type checking failed with %d error(s)", len(errs))
	}

	return codegen.Generate(prog), nil
}

func runFile(filename string) error {
	goCode, err := compile(filename)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "vyr-run-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(goCode), 0644); err != nil {
		return fmt.Errorf("write generated code: %w", err)
	}

	cmd := exec.Command("go", "run", mainFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func buildFile(filename, output string) error {
	goCode, err := compile(filename)
	if err != nil {
		return err
	}

	if output == "" {
		base := filepath.Base(filename)
		output = strings.TrimSuffix(base, filepath.Ext(base))
	}

	tmpDir, err := os.MkdirTemp("", "vyr-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(goCode), 0644); err != nil {
		return fmt.Errorf("write generated code: %w", err)
	}

	absOutput, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", absOutput, mainFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
