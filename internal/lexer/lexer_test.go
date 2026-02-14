package lexer

import (
	"testing"
)

func TestTokenizeHelloWorld(t *testing.T) {
	input := `fn main() {
  "hello world" |> uppercase |> print
}`
	lex := New(input)
	tokens := lex.Tokenize()

	expected := []struct {
		typ TokenType
		lit string
	}{
		{FN, "fn"},
		{IDENT, "main"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{STRING, "hello world"},
		{PIPE, "|>"},
		{IDENT, "uppercase"},
		{PIPE, "|>"},
		{IDENT, "print"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d\ntokens: %v", len(tokens), len(expected), tokens)
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.typ || tokens[i].Literal != exp.lit {
			t.Errorf("token[%d] = %s, want %s(%q)", i, tokens[i], exp.typ, exp.lit)
		}
	}
}

func TestTokenizeEmptyInput(t *testing.T) {
	tokens := New("").Tokenize()
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("empty input should produce single EOF, got %v", tokens)
	}
}

func TestTokenizeIntegers(t *testing.T) {
	tokens := New("42 |> print").Tokenize()
	expected := []struct {
		typ TokenType
		lit string
	}{
		{INT, "42"},
		{PIPE, "|>"},
		{IDENT, "print"},
		{EOF, ""},
	}
	for i, exp := range expected {
		if tokens[i].Type != exp.typ || tokens[i].Literal != exp.lit {
			t.Errorf("token[%d] = %s, want %s(%q)", i, tokens[i], exp.typ, exp.lit)
		}
	}
}

func TestTokenizeWithComments(t *testing.T) {
	input := `// this is a comment
"hello" |> print // trailing comment`
	tokens := New(input).Tokenize()
	expected := []struct {
		typ TokenType
		lit string
	}{
		{STRING, "hello"},
		{PIPE, "|>"},
		{IDENT, "print"},
		{EOF, ""},
	}
	if len(tokens) != len(expected) {
		t.Fatalf("got %d tokens, want %d\ntokens: %v", len(tokens), len(expected), tokens)
	}
	for i, exp := range expected {
		if tokens[i].Type != exp.typ || tokens[i].Literal != exp.lit {
			t.Errorf("token[%d] = %s, want %s(%q)", i, tokens[i], exp.typ, exp.lit)
		}
	}
}
