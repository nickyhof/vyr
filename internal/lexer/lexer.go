package lexer

import "unicode"

// Lexer tokenizes Vyr source code.
type Lexer struct {
	input       []rune
	pos         int
	line        int
	col         int
	tokens      []Token
	interpDepth int // tracks nesting depth of ${...} interpolations
}

// New creates a Lexer for the given source text.
func New(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0, line: 1, col: 1}
}

// Tokenize scans the full input and returns all tokens.
func (l *Lexer) Tokenize() []Token {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return l.tokens
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return Token{Type: EOF, Literal: "", Line: l.line, Col: l.col}
	}

	ch := l.input[l.pos]
	sl, sc := l.line, l.col

	switch {
	case ch == '/' && l.peek() == '/':
		l.skipLineComment()
		return l.nextToken()
	case ch == '"':
		return l.readString(sl, sc)

	// Two-char operators (must come before single-char versions)
	case ch == '|' && l.peek() == '>':
		l.advance()
		l.advance()
		return Token{Type: PIPE, Literal: "|>", Line: sl, Col: sc}
	case ch == '=' && l.peek() == '>':
		l.advance()
		l.advance()
		return Token{Type: ARROW, Literal: "=>", Line: sl, Col: sc}
	case ch == '=' && l.peek() == '=':
		l.advance()
		l.advance()
		return Token{Type: EQ, Literal: "==", Line: sl, Col: sc}
	case ch == '!' && l.peek() == '=':
		l.advance()
		l.advance()
		return Token{Type: NEQ, Literal: "!=", Line: sl, Col: sc}
	case ch == '<' && l.peek() == '=':
		l.advance()
		l.advance()
		return Token{Type: LTE, Literal: "<=", Line: sl, Col: sc}
	case ch == '>' && l.peek() == '=':
		l.advance()
		l.advance()
		return Token{Type: GTE, Literal: ">=", Line: sl, Col: sc}

	// Single-char operators & delimiters
	case ch == '=':
		l.advance()
		return Token{Type: EQUAL, Literal: "=", Line: sl, Col: sc}
	case ch == '<':
		l.advance()
		return Token{Type: LT, Literal: "<", Line: sl, Col: sc}
	case ch == '>':
		l.advance()
		return Token{Type: GT, Literal: ">", Line: sl, Col: sc}
	case ch == '(':
		l.advance()
		return Token{Type: LPAREN, Literal: "(", Line: sl, Col: sc}
	case ch == ')':
		l.advance()
		return Token{Type: RPAREN, Literal: ")", Line: sl, Col: sc}
	case ch == '{':
		l.advance()
		return Token{Type: LBRACE, Literal: "{", Line: sl, Col: sc}
	case ch == '}':
		// If we're inside an interpolation, this closes it
		if l.interpDepth > 0 {
			l.interpDepth--
			l.advance()
			// Continue reading the rest of the string after the interpolation
			return l.readStringContinuation(sl, sc)
		}
		l.advance()
		return Token{Type: RBRACE, Literal: "}", Line: sl, Col: sc}
	case ch == '[':
		l.advance()
		return Token{Type: LBRACKET, Literal: "[", Line: sl, Col: sc}
	case ch == ']':
		l.advance()
		return Token{Type: RBRACKET, Literal: "]", Line: sl, Col: sc}
	case ch == ',':
		l.advance()
		return Token{Type: COMMA, Literal: ",", Line: sl, Col: sc}
	case ch == ':':
		l.advance()
		return Token{Type: COLON, Literal: ":", Line: sl, Col: sc}
	case ch == '+':
		l.advance()
		return Token{Type: PLUS, Literal: "+", Line: sl, Col: sc}
	case ch == '-':
		l.advance()
		return Token{Type: MINUS, Literal: "-", Line: sl, Col: sc}
	case ch == '*':
		l.advance()
		return Token{Type: STAR, Literal: "*", Line: sl, Col: sc}
	case ch == '/' && l.peek() != '/':
		l.advance()
		return Token{Type: SLASH, Literal: "/", Line: sl, Col: sc}
	case ch == '#' && l.peek() == '{':
		l.advance()
		l.advance()
		return Token{Type: HASH_LBRACE, Literal: "#{", Line: sl, Col: sc}
	case ch == '.':
		l.advance()
		return Token{Type: DOT, Literal: ".", Line: sl, Col: sc}

	case unicode.IsDigit(ch):
		return l.readInt(sl, sc)
	case isIdentStart(ch):
		return l.readIdent(sl, sc)
	default:
		l.advance()
		return Token{Type: ILLEGAL, Literal: string(ch), Line: sl, Col: sc}
	}
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.advance()
	}
}

func (l *Lexer) skipLineComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}
}

// readStringBody reads string content until it hits a closing quote or an
// interpolation start (${). It returns the accumulated text, and whether it
// stopped due to interpolation (true) or closing quote (false).
func (l *Lexer) readStringBody() (string, bool) {
	var buf []rune
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			// Escape \${ to produce literal ${
			if next == '$' {
				l.advance() // skip backslash
				buf = append(buf, '$')
				l.advance() // skip $
				continue
			}
			l.advance() // skip backslash
			switch l.input[l.pos] {
			case 'n':
				buf = append(buf, '\n')
			case 't':
				buf = append(buf, '\t')
			case '\\':
				buf = append(buf, '\\')
			case '"':
				buf = append(buf, '"')
			default:
				buf = append(buf, '\\', l.input[l.pos])
			}
			l.advance()
			continue
		}
		// Check for interpolation start: ${
		if l.input[l.pos] == '$' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '{' {
			return string(buf), true
		}
		buf = append(buf, l.input[l.pos])
		l.advance()
	}
	return string(buf), false
}

func (l *Lexer) readString(line, col int) Token {
	l.advance() // skip opening "

	text, isInterp := l.readStringBody()

	if !isInterp {
		// Simple string, no interpolation
		if l.pos < len(l.input) {
			l.advance() // skip closing "
		}
		return Token{Type: STRING, Literal: text, Line: line, Col: col}
	}

	// Interpolation found: emit the leading text as STRING, then INTERP_START
	l.tokens = append(l.tokens, Token{Type: STRING, Literal: text, Line: line, Col: col})

	// Skip past ${
	l.advance() // $
	l.advance() // {
	l.interpDepth++

	return Token{Type: INTERP_START, Literal: "${", Line: l.line, Col: l.col - 2}
}

// readStringContinuation is called after a } closes an interpolation.
// It continues reading the string from where it left off.
func (l *Lexer) readStringContinuation(line, col int) Token {
	// First emit INTERP_END
	l.tokens = append(l.tokens, Token{Type: INTERP_END, Literal: "}", Line: line, Col: col})

	text, isInterp := l.readStringBody()

	if !isInterp {
		// End of string
		if l.pos < len(l.input) {
			l.advance() // skip closing "
		}
		return Token{Type: STRING, Literal: text, Line: l.line, Col: l.col}
	}

	// Another interpolation
	l.tokens = append(l.tokens, Token{Type: STRING, Literal: text, Line: l.line, Col: l.col})

	// Skip past ${
	l.advance() // $
	l.advance() // {
	l.interpDepth++

	return Token{Type: INTERP_START, Literal: "${", Line: l.line, Col: l.col - 2}
}

func (l *Lexer) readInt(line, col int) Token {
	start := l.pos
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.advance()
	}
	return Token{Type: INT, Literal: string(l.input[start:l.pos]), Line: line, Col: col}
}

var keywords = map[string]TokenType{
	"fn": FN, "let": LET, "match": MATCH,
	"true": TRUE, "false": FALSE,
	"if": IF, "else": ELSE,
	"import": IMPORT,
}

func (l *Lexer) readIdent(line, col int) Token {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.advance()
	}
	lit := string(l.input[start:l.pos])
	typ := IDENT
	if kw, ok := keywords[lit]; ok {
		typ = kw
	}
	return Token{Type: typ, Literal: lit, Line: line, Col: col}
}

func isIdentStart(ch rune) bool { return unicode.IsLetter(ch) || ch == '_' }
func isIdentPart(ch rune) bool  { return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' }
