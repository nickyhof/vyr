package parser

import (
	"testing"

	"github.com/nickyhof/vyr/internal/lexer"
)

func TestParseHelloWorld(t *testing.T) {
	input := `fn main() {
  "hello world" |> uppercase |> print
}`
	tokens := lexer.New(input).Tokenize()
	prog, err := New(tokens).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(prog.Decls))
	}

	fn, ok := prog.Decls[0].(*FnDecl)
	if !ok {
		t.Fatalf("expected FnDecl, got %T", prog.Decls[0])
	}
	if fn.Name != "main" {
		t.Errorf("function name = %q, want %q", fn.Name, "main")
	}
	if len(fn.Body) != 1 {
		t.Fatalf("expected 1 body expression, got %d", len(fn.Body))
	}

	// After desugaring: print(uppercase("hello world"))
	outer, ok := fn.Body[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", fn.Body[0])
	}
	outerFn, ok := outer.Fn.(*Ident)
	if !ok || outerFn.Name != "print" {
		t.Errorf("outer call fn = %v, want Ident(print)", outer.Fn)
	}
	if len(outer.Args) != 1 {
		t.Fatalf("outer call args = %d, want 1", len(outer.Args))
	}

	inner, ok := outer.Args[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected inner CallExpr, got %T", outer.Args[0])
	}
	innerFn, ok := inner.Fn.(*Ident)
	if !ok || innerFn.Name != "uppercase" {
		t.Errorf("inner call fn = %v, want Ident(uppercase)", inner.Fn)
	}
	if len(inner.Args) != 1 {
		t.Fatalf("inner call args = %d, want 1", len(inner.Args))
	}

	str, ok := inner.Args[0].(*StringLit)
	if !ok || str.Value != "hello world" {
		t.Errorf("innermost arg = %v, want StringLit(hello world)", inner.Args[0])
	}
}

func TestParsePipeIntoCallWithArgs(t *testing.T) {
	// "hello" |> replace("l", "r") should desugar to replace("hello", "l", "r")
	input := `fn main() {
  "hello" |> replace("l", "r")
}`
	tokens := lexer.New(input).Tokenize()
	prog, err := New(tokens).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	fn := prog.Decls[0].(*FnDecl)
	call, ok := fn.Body[0].(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", fn.Body[0])
	}

	fnIdent := call.Fn.(*Ident)
	if fnIdent.Name != "replace" {
		t.Errorf("fn = %q, want %q", fnIdent.Name, "replace")
	}
	if len(call.Args) != 3 {
		t.Fatalf("args = %d, want 3 (piped value + 2 explicit)", len(call.Args))
	}

	// First arg should be the piped "hello"
	first, ok := call.Args[0].(*StringLit)
	if !ok || first.Value != "hello" {
		t.Errorf("first arg = %v, want StringLit(hello)", call.Args[0])
	}
}
