package lexer

import "fmt"

// TokenType distinguishes the kinds of tokens in Vyr source.
type TokenType int

const (
	// Special
	EOF     TokenType = iota
	ILLEGAL           // unexpected character

	// Literals & identifiers
	IDENT  // variable/function name
	STRING // "hello"
	INT    // 42

	// Keywords
	FN     // fn
	LET    // let
	MATCH  // match
	TRUE   // true
	FALSE  // false
	IMPORT // import
	IF     // if
	ELSE   // else
	WHILE  // while
	MUT    // mut
	STRUCT // struct
	RETURN // return

	// Operators
	PIPE  // |>
	PLUS  // +
	MINUS // -
	STAR  // *
	SLASH // /
	EQUAL // =
	ARROW // =>
	EQ    // ==
	NEQ   // !=
	LT    // <
	GT    // >
	LTE   // <=
	GTE   // >=

	// Delimiters
	LPAREN       // (
	RPAREN       // )
	LBRACE       // {
	RBRACE       // }
	LBRACKET     // [
	RBRACKET     // ]
	COMMA        // ,
	COLON        // :
	DOT          // .
	HASH_LBRACE  // #{
	INTERP_START // ${ (start of interpolation)
	INTERP_END   // } (end of interpolation)
)

var tokenNames = map[TokenType]string{
	EOF: "EOF", ILLEGAL: "ILLEGAL",
	IDENT: "IDENT", STRING: "STRING", INT: "INT",
	FN: "FN", LET: "LET", MATCH: "MATCH",
	TRUE: "TRUE", FALSE: "FALSE", IF: "IF", ELSE: "ELSE", IMPORT: "IMPORT",
	WHILE: "WHILE", MUT: "MUT", STRUCT: "STRUCT", RETURN: "RETURN",
	PIPE: "PIPE", PLUS: "PLUS", MINUS: "MINUS", STAR: "STAR", SLASH: "SLASH",
	EQUAL: "EQUAL", ARROW: "ARROW",
	EQ: "EQ", NEQ: "NEQ", LT: "LT", GT: "GT", LTE: "LTE", GTE: "GTE",
	LPAREN: "LPAREN", RPAREN: "RPAREN",
	LBRACE: "LBRACE", RBRACE: "RBRACE",
	LBRACKET: "LBRACKET", RBRACKET: "RBRACKET",
	COMMA: "COMMA", COLON: "COLON",
	DOT: "DOT", HASH_LBRACE: "HASH_LBRACE",
	INTERP_START: "INTERP_START", INTERP_END: "INTERP_END",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token represents a single lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q)", t.Type, t.Literal)
}
