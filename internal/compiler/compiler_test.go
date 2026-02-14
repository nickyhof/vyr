package compiler

import (
	"testing"

	"github.com/nickyhof/vyr/internal/lexer"
	"github.com/nickyhof/vyr/internal/parser"
)

func compile(t *testing.T, input string) *Bytecode {
	t.Helper()
	tokens := lexer.New(input).Tokenize()
	prog, err := parser.New(tokens).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	code, err := New().Compile(prog)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return code
}

func TestCompileHelloWorld(t *testing.T) {
	code := compile(t, `fn main() {
  "hello world" |> uppercase |> print
}`)
	if code.Entry == nil {
		t.Fatal("no entry function")
	}
	if code.Entry.Name != "main" {
		t.Errorf("entry name = %q, want %q", code.Entry.Name, "main")
	}
	last := code.Entry.Instructions[len(code.Entry.Instructions)-1]
	if last.Op != OpReturn {
		t.Errorf("last instruction = %s, want OpReturn", last.Op)
	}
}

func TestCompileUserFunction(t *testing.T) {
	code := compile(t, `fn double(x: int): int {
  x * 2
}
fn main() {
  21 |> double |> print
}`)
	if len(code.Functions) != 1 {
		t.Fatalf("functions = %d, want 1", len(code.Functions))
	}
	if code.Functions[0].Name != "double" {
		t.Errorf("function name = %q, want %q", code.Functions[0].Name, "double")
	}
	if code.Functions[0].NumParams != 1 {
		t.Errorf("num params = %d, want 1", code.Functions[0].NumParams)
	}
}

func TestCompileLetBinding(t *testing.T) {
	code := compile(t, `fn main() {
  let x = 10 + 5
  x |> print
}`)
	if code.Entry.NumLocals != 1 {
		t.Errorf("main num locals = %d, want 1", code.Entry.NumLocals)
	}
}

func TestCompileIfElse(t *testing.T) {
	code := compile(t, `fn main() {
  if true { 1 } else { 0 }
}`)
	if code.Entry == nil {
		t.Fatal("no entry function")
	}
	// Should contain jump instructions
	hasJump := false
	for _, inst := range code.Entry.Instructions {
		if inst.Op == OpJumpIfNot || inst.Op == OpJump {
			hasJump = true
			break
		}
	}
	if !hasJump {
		t.Error("expected jump instructions for if/else")
	}
}
