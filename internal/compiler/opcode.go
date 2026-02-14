package compiler

import "fmt"

// Opcode represents a single bytecode instruction.
type Opcode byte

const (
	OpConstant   Opcode = iota // Push constant from pool
	OpCall                     // Call function with N args
	OpReturn                   // Return from function
	OpPop                      // Discard top of stack
	OpGetBuiltin               // Push builtin function
	OpGetLocal                 // Push local variable
	OpSetLocal                 // Pop and store local
	OpGetFunc                  // Push compiled function
	OpAdd                      // Pop two, push sum (int or string)
	OpSub                      // Pop two, push difference
	OpMul                      // Pop two, push product
	OpDiv                      // Pop two, push quotient
	OpArray                    // Pop N values, create array
	OpFanOut                   // Fan-out: pop N fns + 1 value, call each
	OpMap                      // Pop fn + array, apply fn to each
	OpFilter                   // Pop fn + array, keep where fn returns true
	OpReduce                   // Pop fn + init + array, fold left
	OpDup                      // Duplicate top of stack
	OpEqual                    // Pop two, push bool (internal equal for match)
	OpJumpIfNot                // Pop bool, jump if false
	OpJump                     // Unconditional jump
	OpEq                       // Pop two, push bool (==)
	OpNeq                      // Pop two, push bool (!=)
	OpLt                       // Pop two, push bool (<)
	OpGt                       // Pop two, push bool (>)
	OpLte                      // Pop two, push bool (<=)
	OpGte                      // Pop two, push bool (>=)
	OpClosure                  // Pop N captures, create closure with function at Operand
	OpGetCapture               // Push captured variable at Operand index
	OpHash                     // Pop N*2 values (key, val pairs), create hashmap
)

var opcodeNames = map[Opcode]string{
	OpConstant: "OpConstant", OpCall: "OpCall", OpReturn: "OpReturn",
	OpPop: "OpPop", OpGetBuiltin: "OpGetBuiltin", OpGetLocal: "OpGetLocal",
	OpSetLocal: "OpSetLocal", OpGetFunc: "OpGetFunc",
	OpAdd: "OpAdd", OpSub: "OpSub", OpMul: "OpMul", OpDiv: "OpDiv",
	OpArray: "OpArray", OpFanOut: "OpFanOut", OpMap: "OpMap",
	OpFilter: "OpFilter", OpReduce: "OpReduce",
	OpDup: "OpDup", OpEqual: "OpEqual", OpJumpIfNot: "OpJumpIfNot", OpJump: "OpJump",
	OpEq: "OpEq", OpNeq: "OpNeq", OpLt: "OpLt", OpGt: "OpGt", OpLte: "OpLte", OpGte: "OpGte",
	OpClosure: "OpClosure", OpGetCapture: "OpGetCapture",
	OpHash: "OpHash",
}

func (o Opcode) String() string {
	if name, ok := opcodeNames[o]; ok {
		return name
	}
	return fmt.Sprintf("Opcode(%d)", int(o))
}

// Instruction is a bytecode instruction with its operand.
type Instruction struct {
	Op      Opcode
	Operand int
}

func (i Instruction) String() string { return fmt.Sprintf("%s %d", i.Op, i.Operand) }

// CompiledFunction represents a compiled user-defined function.
type CompiledFunction struct {
	Name         string
	NumParams    int
	NumLocals    int
	NumCaptures  int // number of captured variables from enclosing scope
	Instructions []Instruction
}

// Closure wraps a CompiledFunction with captured values from its enclosing scope.
type Closure struct {
	Fn       *CompiledFunction
	Captures []interface{}
}

// Result represents a success (Ok) or failure (Err) value.
type Result struct {
	Ok    bool
	Value interface{} // the ok value or the error message string
}

// Bytecode is the compiled output of a program.
type Bytecode struct {
	Entry     *CompiledFunction
	Functions []*CompiledFunction
	Constants []interface{}
}
