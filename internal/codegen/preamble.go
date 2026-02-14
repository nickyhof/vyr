package codegen

// emitPreamble writes the Go runtime (imports, helpers, builtins) that every
// generated program needs. It is written once at the top of the output.
func (g *Generator) emitPreamble() {
	g.buf.WriteString(preamble)
}

const preamble = `package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ===== Vyr Runtime =====

// Result represents a success (Ok) or failure (Err) value.
type Result struct {
	Ok    bool
	Value any
}

// ---------- helpers ----------

func formatValue(v any) string {
	if v == nil {
		return "nil"
	}
	switch val := v.(type) {
	case []any:
		parts := make([]string, len(val))
		for i, elem := range val {
			parts[i] = formatValue(elem)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		pairs := make([]string, 0, len(val))
		ks := make([]string, 0, len(val))
		for k := range val {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			pairs = append(pairs, k+": "+formatValue(val[k]))
		}
		return "#{" + strings.Join(pairs, ", ") + "}"
	case *Result:
		if val.Ok {
			return "Ok(" + formatValue(val.Value) + ")"
		}
		return "Err(" + formatValue(val.Value) + ")"
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case string:
		return val != ""
	default:
		return true
	}
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case int64:
		if bv, ok := b.(int64); ok {
			return av == bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	}
	return false
}

func callFn(fn any, args ...any) any {
	return fn.(func(...any) any)(args...)
}

func typeOf(v any) string {
	if v == nil {
		return "nil"
	}
	switch v.(type) {
	case int64:
		return "int"
	case string:
		return "string"
	case bool:
		return "bool"
	case []any:
		return "array"
	case map[string]any:
		return "map"
	case *Result:
		return "result"
	default:
		return "unknown"
	}
}

// ---------- arithmetic ----------

func vAdd(a, b any) any {
	if ai, ok := a.(int64); ok {
		if bi, ok := b.(int64); ok {
			return ai + bi
		}
	}
	if as, ok := a.(string); ok {
		if bs, ok := b.(string); ok {
			return as + bs
		}
	}
	panic(fmt.Sprintf("add: incompatible types %T and %T", a, b))
}

func vSub(a, b any) any {
	return a.(int64) - b.(int64)
}

func vMul(a, b any) any {
	return a.(int64) * b.(int64)
}

func vDiv(a, b any) any {
	return a.(int64) / b.(int64)
}

// ---------- comparison ----------

func vEq(a, b any) any  { return valuesEqual(a, b) }
func vNeq(a, b any) any { return !valuesEqual(a, b) }

func vLt(a, b any) any {
	if ai, ok := a.(int64); ok {
		return ai < b.(int64)
	}
	if as, ok := a.(string); ok {
		return as < b.(string)
	}
	panic(fmt.Sprintf("lt: cannot compare %T", a))
}

func vGt(a, b any) any {
	if ai, ok := a.(int64); ok {
		return ai > b.(int64)
	}
	if as, ok := a.(string); ok {
		return as > b.(string)
	}
	panic(fmt.Sprintf("gt: cannot compare %T", a))
}

func vLte(a, b any) any {
	if ai, ok := a.(int64); ok {
		return ai <= b.(int64)
	}
	if as, ok := a.(string); ok {
		return as <= b.(string)
	}
	panic(fmt.Sprintf("lte: cannot compare %T", a))
}

func vGte(a, b any) any {
	if ai, ok := a.(int64); ok {
		return ai >= b.(int64)
	}
	if as, ok := a.(string); ok {
		return as >= b.(string)
	}
	panic(fmt.Sprintf("gte: cannot compare %T", a))
}

// ---------- fan-out ----------

func fanOut(value any, fns ...any) any {
	result := make([]any, len(fns))
	for i, fn := range fns {
		result[i] = callFn(fn, value)
	}
	return result
}

// ---------- builtins ----------

func b_print(args ...any) any {
	fmt.Println(formatValue(args[0]))
	return nil
}

func b_uppercase(args ...any) any {
	return strings.ToUpper(args[0].(string))
}

func b_length(args ...any) any {
	switch v := args[0].(type) {
	case []any:
		return int64(len(v))
	case string:
		return int64(len([]rune(v)))
	}
	panic(fmt.Sprintf("length: expected array or string, got %T", args[0]))
}

func b_head(args ...any) any {
	arr := args[0].([]any)
	if len(arr) == 0 {
		panic("head: empty array")
	}
	return arr[0]
}

func b_tail(args ...any) any {
	arr := args[0].([]any)
	if len(arr) == 0 {
		return []any{}
	}
	result := make([]any, len(arr)-1)
	copy(result, arr[1:])
	return result
}

func b_range(args ...any) any {
	n := args[0].(int64)
	result := make([]any, n)
	for i := int64(0); i < n; i++ {
		result[i] = i
	}
	return result
}

func b_concat(args ...any) any {
	a := args[0].([]any)
	b := args[1].([]any)
	result := make([]any, len(a)+len(b))
	copy(result, a)
	copy(result[len(a):], b)
	return result
}

func b_lowercase(args ...any) any {
	return strings.ToLower(args[0].(string))
}

func b_split(args ...any) any {
	s := args[0].(string)
	sep := args[1].(string)
	parts := strings.Split(s, sep)
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result
}

func b_join(args ...any) any {
	arr := args[0].([]any)
	sep := args[1].(string)
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = formatValue(v)
	}
	return strings.Join(parts, sep)
}

func b_trim(args ...any) any {
	return strings.TrimSpace(args[0].(string))
}

func b_contains(args ...any) any {
	switch haystack := args[0].(type) {
	case string:
		needle, ok := args[1].(string)
		if !ok {
			return false
		}
		return strings.Contains(haystack, needle)
	case []any:
		for _, elem := range haystack {
			if valuesEqual(elem, args[1]) {
				return true
			}
		}
		return false
	}
	panic("contains: first argument must be string or array")
}

func b_replace(args ...any) any {
	return strings.ReplaceAll(args[0].(string), args[1].(string), args[2].(string))
}

func b_starts_with(args ...any) any {
	return strings.HasPrefix(args[0].(string), args[1].(string))
}

func b_ends_with(args ...any) any {
	return strings.HasSuffix(args[0].(string), args[1].(string))
}

func b_substring(args ...any) any {
	runes := []rune(args[0].(string))
	start := args[1].(int64)
	end := args[2].(int64)
	if start < 0 {
		start = 0
	}
	if end > int64(len(runes)) {
		end = int64(len(runes))
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

func b_char_at(args ...any) any {
	runes := []rune(args[0].(string))
	idx := args[1].(int64)
	if idx < 0 || idx >= int64(len(runes)) {
		panic(fmt.Sprintf("char_at: index %d out of bounds (length %d)", idx, len(runes)))
	}
	return string(runes[idx])
}

func b_index(args ...any) any {
	arr := args[0].([]any)
	idx := args[1].(int64)
	if idx < 0 || idx >= int64(len(arr)) {
		panic(fmt.Sprintf("index: %d out of bounds (length %d)", idx, len(arr)))
	}
	return arr[idx]
}

func b_mod(args ...any) any {
	a := args[0].(int64)
	b := args[1].(int64)
	if b == 0 {
		panic("mod: division by zero")
	}
	return a % b
}

func b_to_string(args ...any) any {
	return formatValue(args[0])
}

func b_to_int(args ...any) any {
	switch v := args[0].(type) {
	case int64:
		return v
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("to_int: cannot parse %q as integer", v))
		}
		return n
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	}
	panic(fmt.Sprintf("to_int: cannot convert %T to int", args[0]))
}

func b_sort(args ...any) any {
	arr := args[0].([]any)
	sorted := make([]any, len(arr))
	copy(sorted, arr)
	sort.Slice(sorted, func(i, j int) bool {
		ai, aiOk := sorted[i].(int64)
		aj, ajOk := sorted[j].(int64)
		if aiOk && ajOk {
			return ai < aj
		}
		return formatValue(sorted[i]) < formatValue(sorted[j])
	})
	return sorted
}

func b_slice(args ...any) any {
	arr := args[0].([]any)
	start := args[1].(int64)
	end := args[2].(int64)
	if start < 0 {
		start = 0
	}
	if end > int64(len(arr)) {
		end = int64(len(arr))
	}
	if start >= end {
		return []any{}
	}
	result := make([]any, end-start)
	copy(result, arr[start:end])
	return result
}

func b_reverse(args ...any) any {
	switch v := args[0].(type) {
	case string:
		runes := []rune(v)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	case []any:
		result := make([]any, len(v))
		for i, j := 0, len(v)-1; j >= 0; i, j = i+1, j-1 {
			result[i] = v[j]
		}
		return result
	}
	panic(fmt.Sprintf("reverse: expected string or array, got %T", args[0]))
}

func b_type_of(args ...any) any {
	return typeOf(args[0])
}

func b_get(args ...any) any {
	m := args[0].(map[string]any)
	key := args[1].(string)
	if val, exists := m[key]; exists {
		return val
	}
	if len(args) > 2 {
		return args[2]
	}
	return nil
}

func b_set(args ...any) any {
	m := args[0].(map[string]any)
	key := args[1].(string)
	newMap := make(map[string]any, len(m)+1)
	for k, v := range m {
		newMap[k] = v
	}
	newMap[key] = args[2]
	return newMap
}

func b_keys(args ...any) any {
	m := args[0].(map[string]any)
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	result := make([]any, len(ks))
	for i, k := range ks {
		result[i] = k
	}
	return result
}

func b_values(args ...any) any {
	m := args[0].(map[string]any)
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	result := make([]any, len(ks))
	for i, k := range ks {
		result[i] = m[k]
	}
	return result
}

func b_has_key(args ...any) any {
	m := args[0].(map[string]any)
	key := args[1].(string)
	_, exists := m[key]
	return exists
}

func b_merge(args ...any) any {
	m1 := args[0].(map[string]any)
	m2 := args[1].(map[string]any)
	result := make(map[string]any, len(m1)+len(m2))
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

func b_remove(args ...any) any {
	m := args[0].(map[string]any)
	key := args[1].(string)
	result := make(map[string]any, len(m))
	for k, v := range m {
		if k != key {
			result[k] = v
		}
	}
	return result
}

func b_read_file(args ...any) any {
	data, err := os.ReadFile(args[0].(string))
	if err != nil {
		panic(fmt.Sprintf("read_file: %v", err))
	}
	return string(data)
}

func b_write_file(args ...any) any {
	path := args[0].(string)
	content := args[1].(string)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(fmt.Sprintf("write_file: %v", err))
	}
	return content
}

func b_file_exists(args ...any) any {
	_, err := os.Stat(args[0].(string))
	return err == nil
}

func b_append_file(args ...any) any {
	path := args[0].(string)
	content := args[1].(string)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("append_file: %v", err))
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(content); err != nil {
		panic(fmt.Sprintf("append_file: %v", err))
	}
	return content
}

func b_format(args ...any) any {
	tmpl := args[0].(string)
	result := tmpl
	for i := 1; i < len(args); i++ {
		idx := strings.Index(result, "{}")
		if idx == -1 {
			break
		}
		result = result[:idx] + formatValue(args[i]) + result[idx+2:]
	}
	return result
}

func b_ok(args ...any) any {
	return &Result{Ok: true, Value: args[0]}
}

func b_err(args ...any) any {
	return &Result{Ok: false, Value: args[0]}
}

func b_is_ok(args ...any) any {
	r, ok := args[0].(*Result)
	if !ok {
		return false
	}
	return r.Ok
}

func b_is_err(args ...any) any {
	r, ok := args[0].(*Result)
	if !ok {
		return false
	}
	return !r.Ok
}

func b_unwrap(args ...any) any {
	r := args[0].(*Result)
	if !r.Ok {
		panic(fmt.Sprintf("unwrap called on Err: %v", r.Value))
	}
	return r.Value
}

func b_unwrap_or(args ...any) any {
	r, ok := args[0].(*Result)
	if !ok {
		return args[1]
	}
	if r.Ok {
		return r.Value
	}
	return args[1]
}

func b_try_read_file(args ...any) any {
	path := args[0].(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return &Result{Ok: false, Value: err.Error()}
	}
	return &Result{Ok: true, Value: string(data)}
}

func b_try_to_int(args ...any) any {
	s := args[0].(string)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &Result{Ok: false, Value: fmt.Sprintf("cannot parse '%s' as int", s)}
	}
	return &Result{Ok: true, Value: n}
}

func b_map(args ...any) any {
	arr := args[0].([]any)
	fn := args[1]
	result := make([]any, len(arr))
	for i, v := range arr {
		result[i] = callFn(fn, v)
	}
	return result
}

func b_filter(args ...any) any {
	arr := args[0].([]any)
	fn := args[1]
	var result []any
	for _, v := range arr {
		if isTruthy(callFn(fn, v)) {
			result = append(result, v)
		}
	}
	if result == nil {
		return []any{}
	}
	return result
}

func b_reduce(args ...any) any {
	arr := args[0].([]any)
	init := args[1]
	fn := args[2]
	acc := init
	for _, v := range arr {
		acc = callFn(fn, acc, v)
	}
	return acc
}

func b_push(args ...any) any {
	arr := args[0].([]any)
	result := make([]any, len(arr)+1)
	copy(result, arr)
	result[len(arr)] = args[1]
	return result
}

// Suppress unused import warnings.
var _ = os.ReadFile
var _ = sort.Strings
var _ = strconv.Itoa
var _ = strings.Join

`
