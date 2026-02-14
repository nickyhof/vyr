package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nickyhof/vyr/internal/parser"
)

// builtinSet lists all builtin functions available in the generated runtime.
var builtinSet = map[string]bool{
	"print": true, "uppercase": true, "length": true, "head": true, "tail": true,
	"range": true, "concat": true, "lowercase": true, "split": true, "join": true,
	"trim": true, "contains": true, "replace": true, "starts_with": true,
	"ends_with": true, "substring": true, "char_at": true, "index": true,
	"mod": true, "to_string": true, "to_int": true, "sort": true, "slice": true,
	"reverse": true, "type_of": true, "get": true, "set": true, "keys": true,
	"values": true, "has_key": true, "merge": true, "remove": true,
	"read_file": true, "write_file": true, "file_exists": true, "append_file": true,
	"format": true, "ok": true, "err": true, "is_ok": true, "is_err": true,
	"unwrap": true, "unwrap_or": true, "try_read_file": true, "try_to_int": true,
	"map": true, "filter": true, "reduce": true, "push": true, "args": true,
}

// Generator walks a Vyr AST and emits Go source code.
type Generator struct {
	buf        strings.Builder
	indent     int
	userFuncs  map[string]bool
	scopeStack []map[string]bool // stack of local variable scopes
}

// Generate takes a parsed Vyr program and returns a self-contained Go source file.
func Generate(prog *parser.Program) string {
	g := &Generator{
		userFuncs: make(map[string]bool),
	}

	// First pass: collect user function names and struct names.
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *parser.FnDecl:
			g.userFuncs[d.Name] = true
		case *parser.StructDecl:
			g.userFuncs[d.Name] = true // struct constructors are user functions
		}
	}

	// Emit the self-contained Go file.
	g.emitPreamble()

	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *parser.FnDecl:
			g.emitFnDecl(d)
		case *parser.StructDecl:
			g.emitStructDecl(d)
		}
	}

	g.writeln("")
	g.writeln("func main() {")
	g.writeln("\tfn_main()")
	g.writeln("}")

	return g.buf.String()
}

// ---------- scope helpers ----------

func (g *Generator) pushScope() {
	g.scopeStack = append(g.scopeStack, make(map[string]bool))
}

func (g *Generator) popScope() {
	g.scopeStack = g.scopeStack[:len(g.scopeStack)-1]
}

func (g *Generator) defineLocal(name string) {
	if len(g.scopeStack) > 0 {
		g.scopeStack[len(g.scopeStack)-1][name] = true
	}
}

func (g *Generator) isLocal(name string) bool {
	for i := len(g.scopeStack) - 1; i >= 0; i-- {
		if g.scopeStack[i][name] {
			return true
		}
	}
	return false
}

// ---------- name resolution ----------

func (g *Generator) resolveIdent(name string) string {
	if g.isLocal(name) {
		return "v_" + name
	}
	if g.userFuncs[name] {
		return "fn_" + name
	}
	if builtinSet[name] {
		return "b_" + name
	}
	// Fallback: treat as local (may be a captured variable from outer scope).
	return "v_" + name
}

// ---------- function declaration ----------

func (g *Generator) emitFnDecl(fn *parser.FnDecl) {
	g.writeln("")
	g.writef("func fn_%s(args ...any) any {\n", fn.Name)
	g.indent++
	g.pushScope()

	for i, p := range fn.Params {
		g.defineLocal(p.Name)
		g.writeIndent()
		g.writef("v_%s := args[%d]\n", p.Name, i)
	}

	g.emitBody(fn.Body)
	g.popScope()
	g.indent--
	g.writeln("}")
}

// emitStructDecl generates a constructor function for a struct type.
// struct Point { x, y } → func fn_Point(args ...any) any { return map[string]any{...} }
func (g *Generator) emitStructDecl(s *parser.StructDecl) {
	g.writeln("")
	g.writef("func fn_%s(args ...any) any {\n", s.Name)
	g.indent++
	g.writeIndent()
	g.writef("return map[string]any{")
	g.writef("\"__type\": %s", strconv.Quote(s.Name))
	for i, field := range s.Fields {
		g.writef(", %s: args[%d]", strconv.Quote(field), i)
	}
	g.writef("}\n")
	g.indent--
	g.writeln("}")
}

// emitWhileStmt generates a Go for-loop from a while expression.
func (g *Generator) emitWhileStmt(n *parser.WhileExpr) {
	g.writeIndent()
	g.writef("for isTruthy(%s) {\n", g.exprString(n.Condition))
	g.indent++
	for _, stmt := range n.Body {
		g.emitStatement(stmt)
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
}

// ---------- body emission ----------

func (g *Generator) emitBody(body []parser.Node) {
	if len(body) == 0 {
		g.writeIndent()
		g.writef("return nil\n")
		return
	}
	for i, node := range body {
		isLast := i == len(body)-1
		if isLast {
			g.emitReturnNode(node)
		} else {
			g.emitStatement(node)
		}
	}
}

// emitStatement emits a node whose result is discarded.
func (g *Generator) emitStatement(node parser.Node) {
	switch n := node.(type) {
	case *parser.LetStmt:
		g.writeIndent()
		g.defineLocal(n.Name)
		if n.Mutable {
			g.writef("var v_%s any = %s\n", n.Name, g.exprString(n.Value))
		} else {
			g.writef("v_%s := %s\n", n.Name, g.exprString(n.Value))
		}
	case *parser.AssignStmt:
		g.writeIndent()
		g.writef("v_%s = %s\n", n.Name, g.exprString(n.Value))
	case *parser.WhileExpr:
		g.emitWhileStmt(n)
	case *parser.ReturnStmt:
		g.writeIndent()
		if n.Value == nil {
			g.writef("return nil\n")
		} else {
			g.writef("return %s\n", g.exprString(n.Value))
		}
	case *parser.IfExpr:
		g.emitIfStmt(n, false)
	default:
		g.writeIndent()
		g.writef("_ = %s\n", g.exprString(node))
	}
}

// emitReturnNode emits a node as the return value of a function body.
func (g *Generator) emitReturnNode(node parser.Node) {
	switch n := node.(type) {
	case *parser.LetStmt:
		g.emitStatement(n)
		g.writeIndent()
		g.writef("return nil\n")
	case *parser.AssignStmt:
		g.emitStatement(n)
		g.writeIndent()
		g.writef("return nil\n")
	case *parser.WhileExpr:
		g.emitWhileStmt(n)
		g.writeIndent()
		g.writef("return nil\n")
	case *parser.ReturnStmt:
		g.emitStatement(n) // ReturnStmt emitStatement already emits 'return'
	case *parser.IfExpr:
		g.emitIfStmt(n, true)
	case *parser.MatchExpr:
		g.emitMatchReturn(n)
	default:
		g.writeIndent()
		g.writef("return %s\n", g.exprString(node))
	}
}

// ---------- if expression ----------

func (g *Generator) emitIfStmt(n *parser.IfExpr, isReturn bool) {
	g.writeIndent()
	g.writef("if isTruthy(%s) {\n", g.exprString(n.Condition))
	g.indent++
	if isReturn {
		g.emitBody(n.Then)
	} else {
		for _, s := range n.Then {
			g.emitStatement(s)
		}
	}
	g.indent--
	if len(n.Else) > 0 {
		g.writeIndent()
		g.writef("} else {\n")
		g.indent++
		if isReturn {
			g.emitBody(n.Else)
		} else {
			for _, s := range n.Else {
				g.emitStatement(s)
			}
		}
		g.indent--
	} else if isReturn {
		g.writeIndent()
		g.writef("} else {\n")
		g.indent++
		g.writeIndent()
		g.writef("return nil\n")
		g.indent--
	}
	g.writeIndent()
	g.writef("}\n")
}

func (g *Generator) ifExprString(n *parser.IfExpr) string {
	var b strings.Builder
	b.WriteString("func() any {\n")
	saved := g.buf
	g.buf = strings.Builder{}
	oldIndent := g.indent
	g.indent++
	g.emitIfStmt(n, true)
	inner := g.buf.String()
	g.buf = saved
	g.indent = oldIndent
	b.WriteString(inner)
	for i := 0; i < g.indent; i++ {
		b.WriteByte('\t')
	}
	b.WriteString("}()")
	return b.String()
}

// ---------- match expression ----------

func (g *Generator) emitMatchReturn(n *parser.MatchExpr) {
	subject := g.exprString(n.Subject)
	for _, arm := range n.Arms {
		if isWildcard(arm.Pattern) {
			g.writeIndent()
			g.writef("return %s\n", g.exprString(arm.Body))
			return
		}
		if tp, ok := arm.Pattern.(*parser.TypePattern); ok {
			g.writeIndent()
			g.writef("if _tv, _tvOk := %s.(map[string]any); _tvOk && _tv[\"__type\"] == %s {\n", subject, strconv.Quote(tp.TypeName))
			g.indent++
			g.defineLocal(tp.Binding)
			g.writeIndent()
			g.writef("v_%s := %s\n", tp.Binding, subject)
			g.writeIndent()
			g.writef("return %s\n", g.exprString(arm.Body))
			g.indent--
			g.writeIndent()
			g.writef("}\n")
			continue
		}
		g.writeIndent()
		g.writef("if valuesEqual(%s, %s) {\n", subject, g.exprString(arm.Pattern))
		g.indent++
		g.writeIndent()
		g.writef("return %s\n", g.exprString(arm.Body))
		g.indent--
		g.writeIndent()
		g.writef("}\n")
	}
	g.writeIndent()
	g.writef("return nil\n")
}

func (g *Generator) matchExprString(n *parser.MatchExpr) string {
	var b strings.Builder
	b.WriteString("func() any {\n")
	saved := g.buf
	g.buf = strings.Builder{}
	oldIndent := g.indent
	g.indent++
	g.emitMatchReturn(n)
	inner := g.buf.String()
	g.buf = saved
	g.indent = oldIndent
	b.WriteString(inner)
	for i := 0; i < g.indent; i++ {
		b.WriteByte('\t')
	}
	b.WriteString("}()")
	return b.String()
}

func isWildcard(n parser.Node) bool {
	id, ok := n.(*parser.Ident)
	return ok && id.Name == "_"
}

// ---------- expression stringification ----------

func (g *Generator) exprString(node parser.Node) string {
	switch n := node.(type) {
	case *parser.IntLit:
		return fmt.Sprintf("int64(%d)", n.Value)
	case *parser.StringLit:
		if strings.Contains(n.Value, "\n") {
			// Multi-line: use Go raw string literal
			return "`" + n.Value + "`"
		}
		return strconv.Quote(n.Value)
	case *parser.BoolLit:
		if n.Value {
			return "true"
		}
		return "false"
	case *parser.Ident:
		return g.resolveIdent(n.Name)
	case *parser.BinaryExpr:
		return g.binaryExprString(n)
	case *parser.CallExpr:
		return g.callExprString(n)
	case *parser.ArrayLit:
		return g.arrayLitString(n)
	case *parser.HashLit:
		return g.hashLitString(n)
	case *parser.DotExpr:
		return fmt.Sprintf("b_get(%s, %s)", g.exprString(n.Object), strconv.Quote(n.Key))
	case *parser.FnLit:
		return g.fnLitString(n)
	case *parser.FanOutExpr:
		return g.fanOutString(n)
	case *parser.IfExpr:
		return g.ifExprString(n)
	case *parser.MatchExpr:
		return g.matchExprString(n)
	case *parser.InterpLit:
		return g.interpLitString(n)
	default:
		return fmt.Sprintf("nil /* unknown node: %T */", node)
	}
}

func (g *Generator) binaryExprString(n *parser.BinaryExpr) string {
	left := g.exprString(n.Left)
	right := g.exprString(n.Right)
	switch n.Op {
	case "+":
		return fmt.Sprintf("vAdd(%s, %s)", left, right)
	case "-":
		return fmt.Sprintf("vSub(%s, %s)", left, right)
	case "*":
		return fmt.Sprintf("vMul(%s, %s)", left, right)
	case "/":
		return fmt.Sprintf("vDiv(%s, %s)", left, right)
	case "==":
		return fmt.Sprintf("vEq(%s, %s)", left, right)
	case "!=":
		return fmt.Sprintf("vNeq(%s, %s)", left, right)
	case "<":
		return fmt.Sprintf("vLt(%s, %s)", left, right)
	case ">":
		return fmt.Sprintf("vGt(%s, %s)", left, right)
	case "<=":
		return fmt.Sprintf("vLte(%s, %s)", left, right)
	case ">=":
		return fmt.Sprintf("vGte(%s, %s)", left, right)
	default:
		return fmt.Sprintf("nil /* unknown op: %s */", n.Op)
	}
}

func (g *Generator) callExprString(n *parser.CallExpr) string {
	// Determine the function name/reference and args.
	args := make([]string, len(n.Args))
	for i, a := range n.Args {
		args[i] = g.exprString(a)
	}
	argStr := strings.Join(args, ", ")

	if ident, ok := n.Fn.(*parser.Ident); ok {
		// If the identifier is a local variable, it holds a closure value
		// of type `any`, so we must dispatch through callFn.
		if g.isLocal(ident.Name) {
			resolved := g.resolveIdent(ident.Name)
			if len(args) == 0 {
				return fmt.Sprintf("callFn(%s)", resolved)
			}
			return fmt.Sprintf("callFn(%s, %s)", resolved, argStr)
		}
		resolved := g.resolveIdent(ident.Name)
		return fmt.Sprintf("%s(%s)", resolved, argStr)
	}

	// Dynamic call (lambda or expression that evaluates to a function).
	fnExpr := g.exprString(n.Fn)
	if len(args) == 0 {
		return fmt.Sprintf("callFn(%s)", fnExpr)
	}
	return fmt.Sprintf("callFn(%s, %s)", fnExpr, argStr)
}

func (g *Generator) arrayLitString(n *parser.ArrayLit) string {
	if len(n.Elements) == 0 {
		return "[]any{}"
	}
	elems := make([]string, len(n.Elements))
	for i, e := range n.Elements {
		elems[i] = g.exprString(e)
	}
	return fmt.Sprintf("[]any{%s}", strings.Join(elems, ", "))
}

func (g *Generator) hashLitString(n *parser.HashLit) string {
	if len(n.Pairs) == 0 {
		return "map[string]any{}"
	}
	pairs := make([]string, len(n.Pairs))
	for i, p := range n.Pairs {
		pairs[i] = fmt.Sprintf("%s: %s", strconv.Quote(p.Key), g.exprString(p.Value))
	}
	return fmt.Sprintf("map[string]any{%s}", strings.Join(pairs, ", "))
}

func (g *Generator) fnLitString(n *parser.FnLit) string {
	var b strings.Builder
	b.WriteString("func(args ...any) any {\n")

	// Save generator state and emit body into a temp buffer.
	saved := g.buf
	g.buf = strings.Builder{}
	oldIndent := g.indent
	g.indent++
	g.pushScope()

	for i, p := range n.Params {
		g.defineLocal(p.Name)
		g.writeIndent()
		g.writef("v_%s := args[%d]\n", p.Name, i)
	}
	g.emitBody(n.Body)

	g.popScope()
	inner := g.buf.String()
	g.buf = saved
	g.indent = oldIndent

	b.WriteString(inner)
	for i := 0; i < g.indent; i++ {
		b.WriteByte('\t')
	}
	b.WriteString("}")
	return b.String()
}

func (g *Generator) fanOutString(n *parser.FanOutExpr) string {
	val := g.exprString(n.Value)
	fns := make([]string, len(n.Fns))
	for i, f := range n.Fns {
		fns[i] = g.exprString(f)
	}
	return fmt.Sprintf("fanOut(%s, %s)", val, strings.Join(fns, ", "))
}

func (g *Generator) interpLitString(n *parser.InterpLit) string {
	parts := make([]string, 0, len(n.Parts))
	for _, part := range n.Parts {
		if sl, ok := part.(*parser.StringLit); ok {
			if sl.Value == "" {
				continue
			}
			parts = append(parts, strconv.Quote(sl.Value))
		} else {
			parts = append(parts, fmt.Sprintf("formatValue(%s)", g.exprString(part)))
		}
	}
	if len(parts) == 0 {
		return `""`
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " + ")
}

// ---------- output helpers ----------

func (g *Generator) writeln(s string) {
	g.buf.WriteString(s)
	g.buf.WriteByte('\n')
}

func (g *Generator) writef(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.buf.WriteByte('\t')
	}
}
