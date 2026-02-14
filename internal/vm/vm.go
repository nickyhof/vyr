package vm

import (
	"fmt"
	"strings"

	"github.com/nickyhof/vyr/internal/compiler"
)

type callFrame struct {
	fn       *compiler.CompiledFunction
	ip       int
	basePtr  int
	captures []interface{} // captured values for closures
}

// VM is a stack-based bytecode interpreter for Vyr.
type VM struct {
	code     *compiler.Bytecode
	stack    []interface{}
	frames   []callFrame
	builtins []BuiltinFn
	output   *strings.Builder
}

func New(code *compiler.Bytecode) *VM {
	return &VM{
		code:     code,
		stack:    make([]interface{}, 0, 256),
		builtins: Builtins(nil),
	}
}

func NewWithOutput(code *compiler.Bytecode, out *strings.Builder) *VM {
	return &VM{
		code:     code,
		stack:    make([]interface{}, 0, 256),
		builtins: Builtins(out),
		output:   out,
	}
}

func (vm *VM) Run() error {
	vm.pushFrame(vm.code.Entry, 0, nil)
	for i := 0; i < vm.code.Entry.NumLocals; i++ {
		vm.push(nil)
	}
	return vm.executeFrom(0)
}

func (vm *VM) executeFrom(stopDepth int) error {
	for len(vm.frames) > stopDepth {
		frame := &vm.frames[len(vm.frames)-1]

		if frame.ip >= len(frame.fn.Instructions) {
			vm.popFrame()
			vm.push(nil)
			continue
		}

		inst := frame.fn.Instructions[frame.ip]
		frame.ip++

		switch inst.Op {
		case compiler.OpConstant:
			vm.push(vm.code.Constants[inst.Operand])

		case compiler.OpGetBuiltin:
			vm.push(vm.builtins[inst.Operand])

		case compiler.OpGetLocal:
			vm.push(vm.stack[frame.basePtr+inst.Operand])

		case compiler.OpSetLocal:
			vm.stack[frame.basePtr+inst.Operand] = vm.pop()

		case compiler.OpGetFunc:
			vm.push(vm.code.Functions[inst.Operand])

		case compiler.OpGetCapture:
			vm.push(frame.captures[inst.Operand])

		case compiler.OpClosure:
			fn := vm.code.Functions[inst.Operand]
			captures := make([]interface{}, fn.NumCaptures)
			for i := fn.NumCaptures - 1; i >= 0; i-- {
				captures[i] = vm.pop()
			}
			vm.push(&compiler.Closure{Fn: fn, Captures: captures})

		case compiler.OpCall:
			nargs := inst.Operand
			callee := vm.pop()
			switch fn := callee.(type) {
			case BuiltinFn:
				args := make([]interface{}, nargs)
				for i := nargs - 1; i >= 0; i-- {
					args[i] = vm.pop()
				}
				result, err := fn(args...)
				if err != nil {
					return err
				}
				vm.push(result)
			case *compiler.CompiledFunction:
				if nargs != fn.NumParams {
					return fmt.Errorf("%s: expected %d arguments, got %d", fn.Name, fn.NumParams, nargs)
				}
				basePtr := len(vm.stack) - nargs
				vm.pushFrame(fn, basePtr, nil)
				for i := fn.NumParams; i < fn.NumLocals; i++ {
					vm.push(nil)
				}
			case *compiler.Closure:
				if nargs != fn.Fn.NumParams {
					return fmt.Errorf("%s: expected %d arguments, got %d", fn.Fn.Name, fn.Fn.NumParams, nargs)
				}
				basePtr := len(vm.stack) - nargs
				vm.pushFrame(fn.Fn, basePtr, fn.Captures)
				for i := fn.Fn.NumParams; i < fn.Fn.NumLocals; i++ {
					vm.push(nil)
				}
			default:
				return fmt.Errorf("cannot call non-function: %T", callee)
			}

		case compiler.OpReturn:
			retVal := vm.pop()
			basePtr := frame.basePtr
			vm.popFrame()
			vm.stack = vm.stack[:basePtr]
			vm.push(retVal)

		case compiler.OpPop:
			vm.pop()

		// Arithmetic
		case compiler.OpAdd:
			right, left := vm.pop(), vm.pop()
			if ls, ok := left.(string); ok {
				if rs, ok := right.(string); ok {
					vm.push(ls + rs)
					break
				}
			}
			result, err := vm.intOp("+", left, right)
			if err != nil {
				return err
			}
			vm.push(result)

		case compiler.OpSub:
			right, left := vm.pop(), vm.pop()
			r, err := vm.intOp("-", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		case compiler.OpMul:
			right, left := vm.pop(), vm.pop()
			r, err := vm.intOp("*", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		case compiler.OpDiv:
			right, left := vm.pop(), vm.pop()
			r, err := vm.intOp("/", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		// Comparisons
		case compiler.OpEq:
			right, left := vm.pop(), vm.pop()
			vm.push(valuesEqual(left, right))

		case compiler.OpNeq:
			right, left := vm.pop(), vm.pop()
			vm.push(!valuesEqual(left, right))

		case compiler.OpLt:
			right, left := vm.pop(), vm.pop()
			r, err := vm.compareOp("<", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		case compiler.OpGt:
			right, left := vm.pop(), vm.pop()
			r, err := vm.compareOp(">", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		case compiler.OpLte:
			right, left := vm.pop(), vm.pop()
			r, err := vm.compareOp("<=", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		case compiler.OpGte:
			right, left := vm.pop(), vm.pop()
			r, err := vm.compareOp(">=", left, right)
			if err != nil {
				return err
			}
			vm.push(r)

		// Data structures
		case compiler.OpArray:
			n := inst.Operand
			arr := make([]interface{}, n)
			for i := n - 1; i >= 0; i-- {
				arr[i] = vm.pop()
			}
			vm.push(arr)

		case compiler.OpFanOut:
			nfns := inst.Operand
			fns := make([]interface{}, nfns)
			for i := nfns - 1; i >= 0; i-- {
				fns[i] = vm.pop()
			}
			value := vm.pop()
			results := make([]interface{}, nfns)
			for i, fn := range fns {
				r, err := vm.callValue(fn, []interface{}{value})
				if err != nil {
					return err
				}
				results[i] = r
			}
			vm.push(results)

		case compiler.OpMap:
			fnVal := vm.pop()
			arr, err := vm.popArray("map")
			if err != nil {
				return err
			}
			results := make([]interface{}, len(arr))
			for i, elem := range arr {
				r, err := vm.callValue(fnVal, []interface{}{elem})
				if err != nil {
					return err
				}
				results[i] = r
			}
			vm.push(results)

		case compiler.OpFilter:
			fnVal := vm.pop()
			arr, err := vm.popArray("filter")
			if err != nil {
				return err
			}
			var results []interface{}
			for _, elem := range arr {
				r, err := vm.callValue(fnVal, []interface{}{elem})
				if err != nil {
					return err
				}
				if isTruthy(r) {
					results = append(results, elem)
				}
			}
			if results == nil {
				results = []interface{}{}
			}
			vm.push(results)

		case compiler.OpReduce:
			fnVal := vm.pop()
			init := vm.pop()
			arr, err := vm.popArray("reduce")
			if err != nil {
				return err
			}
			acc := init
			for _, elem := range arr {
				r, err := vm.callValue(fnVal, []interface{}{acc, elem})
				if err != nil {
					return err
				}
				acc = r
			}
			vm.push(acc)

		// Control flow
		case compiler.OpDup:
			if len(vm.stack) == 0 {
				vm.push(nil)
			} else {
				vm.push(vm.stack[len(vm.stack)-1])
			}

		case compiler.OpEqual:
			right, left := vm.pop(), vm.pop()
			vm.push(valuesEqual(left, right))

		case compiler.OpJumpIfNot:
			cond := vm.pop()
			if !isTruthy(cond) {
				frame.ip = inst.Operand
			}

		case compiler.OpJump:
			frame.ip = inst.Operand

		case compiler.OpHash:
			n := inst.Operand
			m := make(map[string]interface{}, n)
			for i := n - 1; i >= 0; i-- {
				val := vm.pop()
				key, _ := vm.pop().(string)
				m[key] = val
			}
			vm.push(m)

		default:
			return fmt.Errorf("unknown opcode: %s", inst.Op)
		}
	}
	return nil
}

func (vm *VM) callValue(fn interface{}, args []interface{}) (interface{}, error) {
	switch f := fn.(type) {
	case BuiltinFn:
		return f(args...)
	case *compiler.CompiledFunction:
		if len(args) != f.NumParams {
			return nil, fmt.Errorf("%s: expected %d arguments, got %d", f.Name, f.NumParams, len(args))
		}
		for _, arg := range args {
			vm.push(arg)
		}
		basePtr := len(vm.stack) - len(args)
		vm.pushFrame(f, basePtr, nil)
		for i := f.NumParams; i < f.NumLocals; i++ {
			vm.push(nil)
		}
		stopDepth := len(vm.frames) - 1
		if err := vm.executeFrom(stopDepth); err != nil {
			return nil, err
		}
		return vm.pop(), nil
	case *compiler.Closure:
		if len(args) != f.Fn.NumParams {
			return nil, fmt.Errorf("%s: expected %d arguments, got %d", f.Fn.Name, f.Fn.NumParams, len(args))
		}
		for _, arg := range args {
			vm.push(arg)
		}
		basePtr := len(vm.stack) - len(args)
		vm.pushFrame(f.Fn, basePtr, f.Captures)
		for i := f.Fn.NumParams; i < f.Fn.NumLocals; i++ {
			vm.push(nil)
		}
		stopDepth := len(vm.frames) - 1
		if err := vm.executeFrom(stopDepth); err != nil {
			return nil, err
		}
		return vm.pop(), nil
	default:
		return nil, fmt.Errorf("cannot call %T as function", fn)
	}
}

func (vm *VM) popArray(context string) ([]interface{}, error) {
	val := vm.pop()
	arr, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s: expected array, got %T", context, val)
	}
	return arr, nil
}

func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	}
	return false
}

func (vm *VM) compareOp(op string, left, right interface{}) (bool, error) {
	li, lok := left.(int64)
	ri, rok := right.(int64)
	if lok && rok {
		switch op {
		case "<":
			return li < ri, nil
		case ">":
			return li > ri, nil
		case "<=":
			return li <= ri, nil
		case ">=":
			return li >= ri, nil
		}
	}
	ls, lok := left.(string)
	rs, rok := right.(string)
	if lok && rok {
		switch op {
		case "<":
			return ls < rs, nil
		case ">":
			return ls > rs, nil
		case "<=":
			return ls <= rs, nil
		case ">=":
			return ls >= rs, nil
		}
	}
	return false, fmt.Errorf("cannot compare %T and %T with %s", left, right, op)
}

func (vm *VM) intOp(op string, left, right interface{}) (int64, error) {
	l, lok := left.(int64)
	r, rok := right.(int64)
	if !lok || !rok {
		return 0, fmt.Errorf("arithmetic %s: expected integers, got %T and %T", op, left, right)
	}
	switch op {
	case "+":
		return l + r, nil
	case "-":
		return l - r, nil
	case "*":
		return l * r, nil
	case "/":
		if r == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return l / r, nil
	}
	return 0, fmt.Errorf("unknown op: %s", op)
}

func (vm *VM) pushFrame(fn *compiler.CompiledFunction, bp int, captures []interface{}) {
	vm.frames = append(vm.frames, callFrame{fn: fn, ip: 0, basePtr: bp, captures: captures})
}
func (vm *VM) popFrame()            { vm.frames = vm.frames[:len(vm.frames)-1] }
func (vm *VM) push(val interface{}) { vm.stack = append(vm.stack, val) }
func (vm *VM) pop() interface{} {
	if len(vm.stack) == 0 {
		return nil
	}
	v := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return v
}
