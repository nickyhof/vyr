package checker

import (
	"fmt"

	"github.com/nickyhof/vyr/internal/parser"
)

// Type represents a Vyr type.
type Type string

const (
	TypeInt    Type = "int"
	TypeString Type = "string"
	TypeBool   Type = "bool"
	TypeArray  Type = "array"
	TypeAny    Type = "any"  // untyped — always compatible
	TypeVoid   Type = "void" // no value (e.g. print)
)

// Error represents a compile-time type error.
type Error struct {
	Message string
}

func (e Error) Error() string { return e.Message }

type funcSig struct {
	params     []Type
	returnType Type
}

// Checker performs compile-time type checking on typed Vyr programs.
type Checker struct {
	errors      []Error
	funcSigs    map[string]funcSig
	builtinSigs map[string]funcSig
	scope       map[string]Type
}

// Check runs the type checker on a program. Returns any type errors found.
// Functions without type annotations are not checked (gradual typing).
func Check(prog *parser.Program) []Error {
	c := &Checker{
		funcSigs:    make(map[string]funcSig),
		builtinSigs: defaultBuiltinSigs(),
	}

	// Register all function signatures
	for _, decl := range prog.Decls {
		fn, ok := decl.(*parser.FnDecl)
		if !ok {
			continue
		}
		sig := funcSig{}
		for _, p := range fn.Params {
			if p.Type != "" {
				sig.params = append(sig.params, Type(p.Type))
			} else {
				sig.params = append(sig.params, TypeAny)
			}
		}
		if fn.ReturnType != "" {
			sig.returnType = Type(fn.ReturnType)
		} else {
			sig.returnType = TypeAny
		}
		c.funcSigs[fn.Name] = sig
	}

	// Check all function bodies — even untyped functions can call typed ones
	for _, decl := range prog.Decls {
		fn, ok := decl.(*parser.FnDecl)
		if !ok {
			continue
		}
		c.checkFn(fn)
	}

	return c.errors
}

func (c *Checker) checkFn(fn *parser.FnDecl) {
	c.scope = make(map[string]Type)
	for _, p := range fn.Params {
		if p.Type != "" {
			c.scope[p.Name] = Type(p.Type)
		} else {
			c.scope[p.Name] = TypeAny
		}
	}

	var bodyType Type
	for _, node := range fn.Body {
		bodyType = c.checkNode(node)
	}

	// Check return type
	if fn.ReturnType != "" && bodyType != TypeAny && bodyType != TypeVoid {
		expected := Type(fn.ReturnType)
		if bodyType != expected {
			c.addError("function %s: return type is %s, but body returns %s", fn.Name, expected, bodyType)
		}
	}
}

func (c *Checker) checkNode(node parser.Node) Type {
	switch n := node.(type) {
	case *parser.IntLit:
		return TypeInt
	case *parser.StringLit:
		return TypeString
	case *parser.BoolLit:
		return TypeBool
	case *parser.ArrayLit:
		for _, elem := range n.Elements {
			c.checkNode(elem)
		}
		return TypeArray

	case *parser.Ident:
		if t, ok := c.scope[n.Name]; ok {
			return t
		}
		if sig, ok := c.funcSigs[n.Name]; ok {
			return sig.returnType
		}
		if sig, ok := c.builtinSigs[n.Name]; ok {
			return sig.returnType
		}
		return TypeAny

	case *parser.LetStmt:
		t := c.checkNode(n.Value)
		c.scope[n.Name] = t
		return TypeVoid

	case *parser.BinaryExpr:
		lt := c.checkNode(n.Left)
		rt := c.checkNode(n.Right)
		return c.checkBinaryOp(n.Op, lt, rt)

	case *parser.CallExpr:
		return c.checkCall(n)

	case *parser.IfExpr:
		condType := c.checkNode(n.Condition)
		if condType != TypeAny && condType != TypeBool {
			c.addError("if condition must be bool, got %s", condType)
		}
		var thenType Type
		for _, node := range n.Then {
			thenType = c.checkNode(node)
		}
		if len(n.Else) > 0 {
			var elseType Type
			for _, node := range n.Else {
				elseType = c.checkNode(node)
			}
			if thenType != TypeAny && elseType != TypeAny && thenType != elseType {
				c.addError("if/else branches must return same type: then is %s, else is %s", thenType, elseType)
			}
		}
		return thenType

	case *parser.MatchExpr:
		if n.Subject != nil {
			c.checkNode(n.Subject)
		}
		var armType Type
		for _, arm := range n.Arms {
			c.checkNode(arm.Pattern)
			t := c.checkNode(arm.Body)
			if armType == "" {
				armType = t
			}
		}
		return armType

	case *parser.FnLit:
		return TypeAny // lambdas are untyped for now

	case *parser.FanOutExpr:
		c.checkNode(n.Value)
		for _, fn := range n.Fns {
			c.checkNode(fn)
		}
		return TypeArray

	case *parser.HashLit:
		for _, pair := range n.Pairs {
			c.checkNode(pair.Value)
		}
		return TypeAny // maps are untyped for now

	case *parser.DotExpr:
		c.checkNode(n.Object)
		return TypeAny // field access returns any
	}
	return TypeAny
}

func (c *Checker) checkBinaryOp(op string, lt, rt Type) Type {
	switch op {
	case "+":
		if lt == TypeString && rt == TypeString {
			return TypeString
		}
		if lt == TypeAny || rt == TypeAny {
			// Can't tell if it's string concat or int add
			return TypeAny
		}
		if lt != TypeInt {
			c.addError("operator +: left operand must be int or string, got %s", lt)
		}
		if rt != TypeInt {
			c.addError("operator +: right operand must be int or string, got %s", rt)
		}
		return TypeInt
	case "-", "*", "/":
		if lt != TypeAny && lt != TypeInt {
			c.addError("operator %s: left operand must be int, got %s", op, lt)
		}
		if rt != TypeAny && rt != TypeInt {
			c.addError("operator %s: right operand must be int, got %s", op, rt)
		}
		return TypeInt
	case "==", "!=", "<", ">", "<=", ">=":
		return TypeBool
	}
	return TypeAny
}

func (c *Checker) checkCall(n *parser.CallExpr) Type {
	ident, ok := n.Fn.(*parser.Ident)
	if !ok {
		// Indirect call—can't type check
		for _, arg := range n.Args {
			c.checkNode(arg)
		}
		return TypeAny
	}

	// Check special forms
	switch ident.Name {
	case "map":
		if len(n.Args) >= 1 {
			c.checkNode(n.Args[0])
		}
		if len(n.Args) >= 2 {
			c.checkNode(n.Args[1])
		}
		return TypeArray
	case "filter":
		if len(n.Args) >= 1 {
			c.checkNode(n.Args[0])
		}
		if len(n.Args) >= 2 {
			c.checkNode(n.Args[1])
		}
		return TypeArray
	case "reduce":
		for _, arg := range n.Args {
			c.checkNode(arg)
		}
		return TypeAny
	}

	// Look up signature
	var sig funcSig
	var found bool
	if sig, found = c.funcSigs[ident.Name]; !found {
		if sig, found = c.builtinSigs[ident.Name]; !found {
			for _, arg := range n.Args {
				c.checkNode(arg)
			}
			return TypeAny
		}
	}

	// Check argument types
	for i, arg := range n.Args {
		argType := c.checkNode(arg)
		if i < len(sig.params) {
			expected := sig.params[i]
			if expected != TypeAny && argType != TypeAny && argType != expected {
				c.addError("%s: argument %d expected %s, got %s", ident.Name, i+1, expected, argType)
			}
		}
	}

	return sig.returnType
}

func (c *Checker) addError(format string, args ...interface{}) {
	c.errors = append(c.errors, Error{Message: fmt.Sprintf(format, args...)})
}

func defaultBuiltinSigs() map[string]funcSig {
	return map[string]funcSig{
		"print":       {params: []Type{TypeAny}, returnType: TypeVoid},
		"uppercase":   {params: []Type{TypeString}, returnType: TypeString},
		"length":      {params: []Type{TypeAny}, returnType: TypeInt},
		"head":        {params: []Type{TypeArray}, returnType: TypeAny},
		"tail":        {params: []Type{TypeArray}, returnType: TypeArray},
		"range":       {params: []Type{TypeInt}, returnType: TypeArray},
		"concat":      {params: []Type{TypeArray, TypeArray}, returnType: TypeArray},
		"lowercase":   {params: []Type{TypeString}, returnType: TypeString},
		"split":       {params: []Type{TypeString, TypeString}, returnType: TypeArray},
		"join":        {params: []Type{TypeArray, TypeString}, returnType: TypeString},
		"trim":        {params: []Type{TypeString}, returnType: TypeString},
		"contains":    {params: []Type{TypeAny, TypeAny}, returnType: TypeBool},
		"replace":     {params: []Type{TypeString, TypeString, TypeString}, returnType: TypeString},
		"starts_with": {params: []Type{TypeString, TypeString}, returnType: TypeBool},
		"ends_with":   {params: []Type{TypeString, TypeString}, returnType: TypeBool},
		"substring":   {params: []Type{TypeString, TypeInt, TypeInt}, returnType: TypeString},
		"char_at":     {params: []Type{TypeString, TypeInt}, returnType: TypeString},
		"index":       {params: []Type{TypeArray, TypeInt}, returnType: TypeAny},
		"mod":         {params: []Type{TypeInt, TypeInt}, returnType: TypeInt},
		"to_string":   {params: []Type{TypeAny}, returnType: TypeString},
		"to_int":      {params: []Type{TypeAny}, returnType: TypeInt},
		"sort":        {params: []Type{TypeArray}, returnType: TypeArray},
		"slice":       {params: []Type{TypeArray, TypeInt, TypeInt}, returnType: TypeArray},
		"reverse":     {params: []Type{TypeAny}, returnType: TypeAny},
		"type_of":     {params: []Type{TypeAny}, returnType: TypeString},
		"get":         {params: []Type{TypeAny, TypeString}, returnType: TypeAny},
		"set":         {params: []Type{TypeAny, TypeString, TypeAny}, returnType: TypeAny},
		"keys":        {params: []Type{TypeAny}, returnType: TypeArray},
		"values":      {params: []Type{TypeAny}, returnType: TypeArray},
		"has_key":     {params: []Type{TypeAny, TypeString}, returnType: TypeBool},
		"merge":       {params: []Type{TypeAny, TypeAny}, returnType: TypeAny},
		"remove":      {params: []Type{TypeAny, TypeString}, returnType: TypeAny},
		"read_file":   {params: []Type{TypeString}, returnType: TypeString},
		"write_file":  {params: []Type{TypeString, TypeString}, returnType: TypeString},

		"file_exists":   {params: []Type{TypeString}, returnType: TypeBool},
		"append_file":   {params: []Type{TypeString, TypeString}, returnType: TypeString},
		"format":        {params: []Type{TypeString}, returnType: TypeString},
		"ok":            {params: []Type{TypeAny}, returnType: TypeAny},
		"err":           {params: []Type{TypeAny}, returnType: TypeAny},
		"is_ok":         {params: []Type{TypeAny}, returnType: TypeBool},
		"is_err":        {params: []Type{TypeAny}, returnType: TypeBool},
		"unwrap":        {params: []Type{TypeAny}, returnType: TypeAny},
		"unwrap_or":     {params: []Type{TypeAny, TypeAny}, returnType: TypeAny},
		"try_read_file": {params: []Type{TypeString}, returnType: TypeAny},
		"try_to_int":    {params: []Type{TypeAny}, returnType: TypeAny},
	}
}
