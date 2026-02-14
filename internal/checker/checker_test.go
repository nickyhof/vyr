package checker

import (
	"testing"

	"github.com/nickyhof/vyr/internal/lexer"
	"github.com/nickyhof/vyr/internal/parser"
)

func check(t *testing.T, input string) []Error {
	t.Helper()
	tokens := lexer.New(input).Tokenize()
	prog, err := parser.New(tokens).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return Check(prog)
}

func TestUntypedFunctionPasses(t *testing.T) {
	errs := check(t, `fn main() {
  "hello" |> print
}`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for untyped function, got %v", errs)
	}
}

func TestTypedFunctionPasses(t *testing.T) {
	errs := check(t, `fn double(x: int): int {
  x * 2
}
fn main() { 42 |> double |> print }`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestArithmeticOnStringError(t *testing.T) {
	errs := check(t, `fn bad(x: string): int {
  x * 2
}
fn main() { "hi" |> bad }`)
	if len(errs) == 0 {
		t.Error("expected type error for string * int")
	}
}

func TestReturnTypeMismatch(t *testing.T) {
	errs := check(t, `fn wrong(x: int): string {
  x * 2
}
fn main() { 42 |> wrong }`)
	if len(errs) == 0 {
		t.Error("expected type error for returning int from string function")
	}
}

func TestCallArgTypeMismatch(t *testing.T) {
	errs := check(t, `fn double(x: int): int {
  x * 2
}
fn main() {
  "hello" |> double |> print
}`)
	if len(errs) == 0 {
		t.Error("expected type error for passing string to int param")
	}
}

func TestIfConditionMustBeBool(t *testing.T) {
	errs := check(t, `fn check(x: int): int {
  if x { 1 } else { 0 }
}
fn main() { 5 |> check }`)
	if len(errs) == 0 {
		t.Error("expected type error for non-bool if condition")
	}
}

func TestStringConcatPasses(t *testing.T) {
	errs := check(t, `fn greet(name: string): string {
  "Hello, " + name
}
fn main() { "world" |> greet |> print }`)
	if len(errs) != 0 {
		t.Errorf("expected no errors for string concat, got %v", errs)
	}
}
