package compiler

import (
	"fmt"

	"github.com/nickyhof/vyr/internal/parser"
)

// builtinNames lists all builtin functions. Index = OpGetBuiltin operand.
var builtinNames = []string{
	"print",         // 0
	"uppercase",     // 1
	"length",        // 2
	"head",          // 3
	"tail",          // 4
	"range",         // 5
	"concat",        // 6
	"lowercase",     // 7
	"split",         // 8
	"join",          // 9
	"trim",          // 10
	"contains",      // 11
	"replace",       // 12
	"starts_with",   // 13
	"ends_with",     // 14
	"substring",     // 15
	"char_at",       // 16
	"index",         // 17
	"mod",           // 18
	"to_string",     // 19
	"to_int",        // 20
	"sort",          // 21
	"slice",         // 22
	"reverse",       // 23
	"type_of",       // 24
	"get",           // 25
	"set",           // 26
	"keys",          // 27
	"values",        // 28
	"has_key",       // 29
	"merge",         // 30
	"remove",        // 31
	"read_file",     // 32
	"write_file",    // 33
	"file_exists",   // 34
	"append_file",   // 35
	"format",        // 36
	"ok",            // 37
	"err",           // 38
	"is_ok",         // 39
	"is_err",        // 40
	"unwrap",        // 41
	"unwrap_or",     // 42
	"try_read_file", // 43
	"try_to_int",    // 44
}

// specialForms lists functions compiled as special opcodes (not regular calls).
var specialForms = map[string]bool{"map": true, "filter": true, "reduce": true}

// captureInfo records a variable captured from an enclosing scope.
type captureInfo struct {
	name          string
	enclosingSlot int  // slot index in the enclosing scope
	fromCapture   bool // true if capturing from enclosing scope's captures (not locals)
}

// scope tracks local variables for the function being compiled.
type scope struct {
	locals         map[string]int
	numLocals      int
	captures       []captureInfo
	captureMap     map[string]int // name → capture index
	enclosingScope *scope         // nil for top-level functions
}

func newScope(params []parser.Param) *scope {
	s := &scope{
		locals:     make(map[string]int),
		captureMap: make(map[string]int),
	}
	for _, p := range params {
		s.locals[p.Name] = s.numLocals
		s.numLocals++
	}
	return s
}

func (s *scope) define(name string) int {
	idx := s.numLocals
	s.locals[name] = idx
	s.numLocals++
	return idx
}

func (s *scope) resolve(name string) (int, bool) {
	idx, ok := s.locals[name]
	return idx, ok
}

// addCapture records a captured variable and returns its capture index.
func (s *scope) addCapture(name string, enclosingSlot int, fromCapture bool) int {
	if idx, ok := s.captureMap[name]; ok {
		return idx // already captured
	}
	idx := len(s.captures)
	s.captures = append(s.captures, captureInfo{
		name: name, enclosingSlot: enclosingSlot, fromCapture: fromCapture,
	})
	s.captureMap[name] = idx
	return idx
}

// Compiler translates a parsed AST into bytecode.
type Compiler struct {
	constants   []interface{}
	functions   []*CompiledFunction
	funcIndex   map[string]int
	lambdaCount int

	currentInstructions []Instruction
	currentScope        *scope
}

// New creates a new Compiler.
func New() *Compiler {
	return &Compiler{funcIndex: make(map[string]int)}
}

// Compile compiles a full program into bytecode.
func (c *Compiler) Compile(prog *parser.Program) (*Bytecode, error) {
	// First pass: register all named function declarations
	for _, decl := range prog.Decls {
		fn, ok := decl.(*parser.FnDecl)
		if !ok {
			return nil, fmt.Errorf("expected function declaration, got %T", decl)
		}
		if fn.Name != "main" {
			idx := len(c.functions)
			c.funcIndex[fn.Name] = idx
			c.functions = append(c.functions, nil)
		}
	}

	// Second pass: compile all functions
	var mainFn *parser.FnDecl
	for _, decl := range prog.Decls {
		fn := decl.(*parser.FnDecl)
		if fn.Name == "main" {
			mainFn = fn
			continue
		}
		compiled, err := c.compileFn(fn.Name, fn.Params, fn.Body)
		if err != nil {
			return nil, err
		}
		c.functions[c.funcIndex[fn.Name]] = compiled
	}

	if mainFn == nil {
		return nil, fmt.Errorf("no main function found")
	}
	entry, err := c.compileFn("main", mainFn.Params, mainFn.Body)
	if err != nil {
		return nil, err
	}

	return &Bytecode{
		Entry:     entry,
		Functions: c.functions,
		Constants: c.constants,
	}, nil
}

func (c *Compiler) compileFn(name string, params []parser.Param, body []parser.Node) (*CompiledFunction, error) {
	prev := c.currentInstructions
	prevScope := c.currentScope

	c.currentInstructions = nil
	c.currentScope = newScope(params)

	if err := c.compileBody(body); err != nil {
		return nil, err
	}
	c.emit(OpReturn, 0)

	compiled := &CompiledFunction{
		Name:         name,
		NumParams:    len(params),
		NumLocals:    c.currentScope.numLocals,
		Instructions: c.currentInstructions,
	}

	c.currentInstructions = prev
	c.currentScope = prevScope
	return compiled, nil
}

func (c *Compiler) compileBody(body []parser.Node) error {
	for i, node := range body {
		if err := c.compileNode(node); err != nil {
			return err
		}
		if i < len(body)-1 {
			if _, isLet := node.(*parser.LetStmt); !isLet {
				c.emit(OpPop, 0)
			}
		}
	}
	return nil
}

func (c *Compiler) compileNode(node parser.Node) error {
	switch n := node.(type) {
	case *parser.StringLit:
		c.emit(OpConstant, c.addConstant(n.Value))

	case *parser.IntLit:
		c.emit(OpConstant, c.addConstant(n.Value))

	case *parser.BoolLit:
		c.emit(OpConstant, c.addConstant(n.Value))

	case *parser.Ident:
		return c.compileIdent(n.Name)

	case *parser.CallExpr:
		return c.compileCall(n)

	case *parser.LetStmt:
		if err := c.compileNode(n.Value); err != nil {
			return err
		}
		slot := c.currentScope.define(n.Name)
		c.emit(OpSetLocal, slot)

	case *parser.BinaryExpr:
		return c.compileBinary(n)

	case *parser.ArrayLit:
		for _, elem := range n.Elements {
			if err := c.compileNode(elem); err != nil {
				return err
			}
		}
		c.emit(OpArray, len(n.Elements))

	case *parser.FnLit:
		return c.compileLambda(n)

	case *parser.FanOutExpr:
		return c.compileFanOut(n)

	case *parser.MatchExpr:
		return c.compileMatch(n)

	case *parser.IfExpr:
		return c.compileIf(n)

	case *parser.HashLit:
		for _, pair := range n.Pairs {
			c.emit(OpConstant, c.addConstant(pair.Key))
			if err := c.compileNode(pair.Value); err != nil {
				return err
			}
		}
		c.emit(OpHash, len(n.Pairs))

	case *parser.DotExpr:
		// Desugar user.name → get(user, "name")
		if err := c.compileNode(n.Object); err != nil {
			return err
		}
		c.emit(OpConstant, c.addConstant(n.Key))
		idx := c.lookupBuiltin("get")
		c.emit(OpGetBuiltin, idx)
		c.emit(OpCall, 2)

	case *parser.InterpLit:
		// Desugar "hello ${name}!" → to_string("hello ") + to_string(name) + to_string("!")
		toStringIdx := c.lookupBuiltin("to_string")
		emitted := 0
		for _, part := range n.Parts {
			// Skip empty string fragments
			if sl, ok := part.(*parser.StringLit); ok && sl.Value == "" {
				continue
			}
			if err := c.compileNode(part); err != nil {
				return err
			}
			// StringLit parts are already strings, no need for to_string
			if _, isStr := part.(*parser.StringLit); !isStr {
				c.emit(OpGetBuiltin, toStringIdx)
				c.emit(OpCall, 1)
			}
			if emitted > 0 {
				c.emit(OpAdd, 0)
			}
			emitted++
		}
		if emitted == 0 {
			// Edge case: all parts were empty
			c.emit(OpConstant, c.addConstant(""))
		}

	default:
		return fmt.Errorf("cannot compile node type: %T", node)
	}
	return nil
}

func (c *Compiler) compileBinary(n *parser.BinaryExpr) error {
	if err := c.compileNode(n.Left); err != nil {
		return err
	}
	if err := c.compileNode(n.Right); err != nil {
		return err
	}
	switch n.Op {
	case "+":
		c.emit(OpAdd, 0)
	case "-":
		c.emit(OpSub, 0)
	case "*":
		c.emit(OpMul, 0)
	case "/":
		c.emit(OpDiv, 0)
	case "==":
		c.emit(OpEq, 0)
	case "!=":
		c.emit(OpNeq, 0)
	case "<":
		c.emit(OpLt, 0)
	case ">":
		c.emit(OpGt, 0)
	case "<=":
		c.emit(OpLte, 0)
	case ">=":
		c.emit(OpGte, 0)
	default:
		return fmt.Errorf("unknown operator: %s", n.Op)
	}
	return nil
}

func (c *Compiler) compileCall(n *parser.CallExpr) error {
	// Special forms
	if ident, ok := n.Fn.(*parser.Ident); ok {
		switch ident.Name {
		case "map":
			if len(n.Args) != 2 {
				return fmt.Errorf("map: expected 2 arguments, got %d", len(n.Args))
			}
			if err := c.compileNode(n.Args[0]); err != nil {
				return err
			}
			if err := c.compileNode(n.Args[1]); err != nil {
				return err
			}
			c.emit(OpMap, 0)
			return nil

		case "filter":
			if len(n.Args) != 2 {
				return fmt.Errorf("filter: expected 2 arguments, got %d", len(n.Args))
			}
			if err := c.compileNode(n.Args[0]); err != nil {
				return err
			}
			if err := c.compileNode(n.Args[1]); err != nil {
				return err
			}
			c.emit(OpFilter, 0)
			return nil

		case "reduce":
			if len(n.Args) != 3 {
				return fmt.Errorf("reduce: expected 3 arguments, got %d", len(n.Args))
			}
			if err := c.compileNode(n.Args[0]); err != nil {
				return err
			}
			if err := c.compileNode(n.Args[1]); err != nil {
				return err
			}
			if err := c.compileNode(n.Args[2]); err != nil {
				return err
			}
			c.emit(OpReduce, 0)
			return nil
		}
	}

	// Regular call
	for _, arg := range n.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}
	if err := c.compileNode(n.Fn); err != nil {
		return err
	}
	c.emit(OpCall, len(n.Args))
	return nil
}

func (c *Compiler) compileLambda(n *parser.FnLit) error {
	c.lambdaCount++
	name := fmt.Sprintf("__lambda_%d", c.lambdaCount)

	// Save enclosing state
	enclosingScope := c.currentScope
	prevInstructions := c.currentInstructions

	// Create lambda scope with enclosing scope reference for captures
	c.currentInstructions = nil
	c.currentScope = &scope{
		locals:         make(map[string]int),
		captureMap:     make(map[string]int),
		enclosingScope: enclosingScope,
	}
	for _, p := range n.Params {
		c.currentScope.locals[p.Name] = c.currentScope.numLocals
		c.currentScope.numLocals++
	}

	// Compile body — identifier resolution will check enclosing scope
	if err := c.compileBody(n.Body); err != nil {
		return err
	}
	c.emit(OpReturn, 0)

	captures := c.currentScope.captures
	compiled := &CompiledFunction{
		Name:         name,
		NumParams:    len(n.Params),
		NumLocals:    c.currentScope.numLocals,
		NumCaptures:  len(captures),
		Instructions: c.currentInstructions,
	}

	// Restore enclosing state
	c.currentInstructions = prevInstructions
	c.currentScope = enclosingScope

	// Register function
	idx := len(c.functions)
	c.functions = append(c.functions, compiled)

	if len(captures) > 0 {
		// Emit in the enclosing scope to push captured values
		for _, cap := range captures {
			if cap.fromCapture {
				c.emit(OpGetCapture, cap.enclosingSlot)
			} else {
				c.emit(OpGetLocal, cap.enclosingSlot)
			}
		}
		c.emit(OpClosure, idx)
	} else {
		c.emit(OpGetFunc, idx)
	}
	return nil
}

func (c *Compiler) compileFanOut(n *parser.FanOutExpr) error {
	if err := c.compileNode(n.Value); err != nil {
		return err
	}
	for _, fn := range n.Fns {
		if err := c.compileNode(fn); err != nil {
			return err
		}
	}
	c.emit(OpFanOut, len(n.Fns))
	return nil
}

func (c *Compiler) compileMatch(n *parser.MatchExpr) error {
	if n.Subject == nil {
		return fmt.Errorf("match expression has no subject")
	}
	if err := c.compileNode(n.Subject); err != nil {
		return err
	}

	var endJumps []int
	for i, arm := range n.Arms {
		if ident, ok := arm.Pattern.(*parser.Ident); ok && ident.Name == "_" {
			c.emit(OpPop, 0)
			if err := c.compileNode(arm.Body); err != nil {
				return err
			}
		} else {
			c.emit(OpDup, 0)
			if err := c.compileNode(arm.Pattern); err != nil {
				return err
			}
			c.emit(OpEqual, 0)
			nextArm := c.emitJump(OpJumpIfNot)
			c.emit(OpPop, 0)
			if err := c.compileNode(arm.Body); err != nil {
				return err
			}
			endJumps = append(endJumps, c.emitJump(OpJump))
			c.patchJump(nextArm)

			if i == len(n.Arms)-1 {
				c.emit(OpPop, 0)
				c.emit(OpConstant, c.addConstant(nil))
			}
		}
	}
	for _, j := range endJumps {
		c.patchJump(j)
	}
	return nil
}

func (c *Compiler) compileIf(n *parser.IfExpr) error {
	if err := c.compileNode(n.Condition); err != nil {
		return err
	}
	elseJump := c.emitJump(OpJumpIfNot)

	if err := c.compileBody(n.Then); err != nil {
		return err
	}

	if len(n.Else) > 0 {
		endJump := c.emitJump(OpJump)
		c.patchJump(elseJump)
		if err := c.compileBody(n.Else); err != nil {
			return err
		}
		c.patchJump(endJump)
	} else {
		endJump := c.emitJump(OpJump)
		c.patchJump(elseJump)
		c.emit(OpConstant, c.addConstant(nil))
		c.patchJump(endJump)
	}
	return nil
}

func (c *Compiler) emitJump(op Opcode) int {
	idx := len(c.currentInstructions)
	c.emit(op, 0)
	return idx
}

func (c *Compiler) patchJump(idx int) {
	c.currentInstructions[idx].Operand = len(c.currentInstructions)
}

func (c *Compiler) compileIdent(name string) error {
	// 1. Current scope's locals
	if slot, ok := c.currentScope.resolve(name); ok {
		c.emit(OpGetLocal, slot)
		return nil
	}

	// 2. Already-captured variables
	if idx, ok := c.currentScope.captureMap[name]; ok {
		c.emit(OpGetCapture, idx)
		return nil
	}

	// 3. Enclosing scope — create a new capture
	if c.currentScope.enclosingScope != nil {
		enc := c.currentScope.enclosingScope
		// Check enclosing locals
		if slot, ok := enc.resolve(name); ok {
			idx := c.currentScope.addCapture(name, slot, false)
			c.emit(OpGetCapture, idx)
			return nil
		}
		// Check enclosing captures (for nested closures)
		if capIdx, ok := enc.captureMap[name]; ok {
			idx := c.currentScope.addCapture(name, capIdx, true)
			c.emit(OpGetCapture, idx)
			return nil
		}
	}

	// 4. Special forms
	if _, isSpecial := specialForms[name]; isSpecial {
		return fmt.Errorf("%s is a special form, not a regular function", name)
	}

	// 5. User functions
	if idx, ok := c.funcIndex[name]; ok {
		c.emit(OpGetFunc, idx)
		return nil
	}

	// 6. Builtins
	if idx := c.lookupBuiltin(name); idx >= 0 {
		c.emit(OpGetBuiltin, idx)
		return nil
	}

	return fmt.Errorf("undefined identifier: %s", name)
}

func (c *Compiler) emit(op Opcode, operand int) {
	c.currentInstructions = append(c.currentInstructions, Instruction{Op: op, Operand: operand})
}

func (c *Compiler) addConstant(val interface{}) int {
	c.constants = append(c.constants, val)
	return len(c.constants) - 1
}

func (c *Compiler) lookupBuiltin(name string) int {
	for i, n := range builtinNames {
		if n == name {
			return i
		}
	}
	return -1
}
