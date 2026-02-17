package main

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

func b_unwrap_err(args ...any) any {
	r := args[0].(*Result)
	if r.Ok {
		panic(fmt.Sprintf("unwrap_err called on Ok: %v", r.Value))
	}
	return r.Value
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

func b_args(_ ...any) any {
	result := make([]any, len(os.Args)-1)
	for i, a := range os.Args[1:] {
		result[i] = a
	}
	return result
}

// Suppress unused import warnings.
var _ = os.ReadFile
var _ = sort.Strings
var _ = strconv.Itoa
var _ = strings.Join


func fn_Token(args ...any) any {
	return map[string]any{"__type": "Token", "type": args[0], "literal": args[1], "line": args[2], "col": args[3]}
}

func fn_is_alpha(args ...any) any {
	v_ch := args[0]
	if isTruthy(vGte(v_ch, "a")) {
		if isTruthy(vLte(v_ch, "z")) {
			return true
		}
	}
	if isTruthy(vGte(v_ch, "A")) {
		if isTruthy(vLte(v_ch, "Z")) {
			return true
		}
	}
	if isTruthy(vEq(v_ch, "_")) {
		return true
	}
	return false
}

func fn_is_digit(args ...any) any {
	v_ch := args[0]
	if isTruthy(vGte(v_ch, "0")) {
		if isTruthy(vLte(v_ch, "9")) {
			return true
		}
	}
	return false
}

func fn_is_ident_char(args ...any) any {
	v_ch := args[0]
	if isTruthy(fn_is_alpha(v_ch)) {
		return true
	}
	return fn_is_digit(v_ch)
}

func fn_is_whitespace(args ...any) any {
	v_ch := args[0]
	if isTruthy(vEq(v_ch, " ")) {
		return true
	}
	if isTruthy(vEq(v_ch, `
`)) {
		return true
	}
	if isTruthy(vEq(v_ch, "\t")) {
		return true
	}
	if isTruthy(vEq(v_ch, "\\r")) {
		return true
	}
	return false
}

func fn_keyword_type(args ...any) any {
	v_word := args[0]
	if valuesEqual(v_word, "fn") {
		return "FN"
	}
	if valuesEqual(v_word, "let") {
		return "LET"
	}
	if valuesEqual(v_word, "match") {
		return "MATCH"
	}
	if valuesEqual(v_word, "true") {
		return "TRUE"
	}
	if valuesEqual(v_word, "false") {
		return "FALSE"
	}
	if valuesEqual(v_word, "if") {
		return "IF"
	}
	if valuesEqual(v_word, "else") {
		return "ELSE"
	}
	if valuesEqual(v_word, "import") {
		return "IMPORT"
	}
	if valuesEqual(v_word, "while") {
		return "WHILE"
	}
	if valuesEqual(v_word, "mut") {
		return "MUT"
	}
	if valuesEqual(v_word, "struct") {
		return "STRUCT"
	}
	if valuesEqual(v_word, "return") {
		return "RETURN"
	}
	return "IDENT"
}

func fn_s_ch(args ...any) any {
	v_s := args[0]
	return b_char_at(b_get(v_s, "input"), b_get(v_s, "pos"))
}

func fn_s_at(args ...any) any {
	v_s := args[0]
	v_n := args[1]
	return b_char_at(b_get(v_s, "input"), v_n)
}

func fn_s_pos(args ...any) any {
	v_s := args[0]
	return b_get(v_s, "pos")
}

func fn_s_len(args ...any) any {
	v_s := args[0]
	return b_get(v_s, "len")
}

func fn_s_line(args ...any) any {
	v_s := args[0]
	return b_get(v_s, "line")
}

func fn_s_col(args ...any) any {
	v_s := args[0]
	return b_get(v_s, "col")
}

func fn_s_eof(args ...any) any {
	v_s := args[0]
	return vGte(fn_s_pos(v_s), fn_s_len(v_s))
}

func fn_s_advance(args ...any) any {
	v_s := args[0]
	v_p := fn_s_pos(v_s)
	if isTruthy(vGte(v_p, fn_s_len(v_s))) {
		return v_s
	}
	v_ch := fn_s_at(v_s, v_p)
	if isTruthy(vEq(v_ch, `
`)) {
		return b_set(b_set(b_set(v_s, "pos", vAdd(v_p, int64(1))), "line", vAdd(fn_s_line(v_s), int64(1))), "col", int64(1))
	} else {
		return b_set(b_set(v_s, "pos", vAdd(v_p, int64(1))), "col", vAdd(fn_s_col(v_s), int64(1)))
	}
}

func fn_s_peek(args ...any) any {
	v_s := args[0]
	v_p := vAdd(fn_s_pos(v_s), int64(1))
	if isTruthy(vLt(v_p, fn_s_len(v_s))) {
		return fn_s_at(v_s, v_p)
	}
	return ""
}

func fn_s_emit(args ...any) any {
	v_s := args[0]
	v_tok := args[1]
	return b_set(v_s, "tokens", b_push(b_get(v_s, "tokens"), v_tok))
}

func fn_skip_whitespace(args ...any) any {
	v_s := args[0]
	if isTruthy(fn_s_eof(v_s)) {
		return v_s
	}
	if isTruthy(vEq(fn_is_whitespace(fn_s_ch(v_s)), false)) {
		return v_s
	}
	return fn_skip_whitespace(fn_s_advance(v_s))
}

func fn_skip_line_comment(args ...any) any {
	v_s := args[0]
	if isTruthy(fn_s_eof(v_s)) {
		return v_s
	}
	if isTruthy(vEq(fn_s_ch(v_s), `
`)) {
		return v_s
	}
	return fn_skip_line_comment(fn_s_advance(v_s))
}

func fn_read_int(args ...any) any {
	v_s := args[0]
	v_sl := args[1]
	v_sc := args[2]
	v_buf := args[3]
	if isTruthy(fn_s_eof(v_s)) {
		return fn_s_emit(v_s, fn_Token("INT", v_buf, v_sl, v_sc))
	}
	if isTruthy(fn_is_digit(fn_s_ch(v_s))) {
		return fn_read_int(fn_s_advance(v_s), v_sl, v_sc, vAdd(v_buf, fn_s_ch(v_s)))
	}
	return fn_s_emit(v_s, fn_Token("INT", v_buf, v_sl, v_sc))
}

func fn_read_ident(args ...any) any {
	v_s := args[0]
	v_sl := args[1]
	v_sc := args[2]
	v_buf := args[3]
	if isTruthy(fn_s_eof(v_s)) {
		return fn_s_emit(v_s, fn_Token(fn_keyword_type(v_buf), v_buf, v_sl, v_sc))
	}
	if isTruthy(fn_is_ident_char(fn_s_ch(v_s))) {
		return fn_read_ident(fn_s_advance(v_s), v_sl, v_sc, vAdd(v_buf, fn_s_ch(v_s)))
	}
	return fn_s_emit(v_s, fn_Token(fn_keyword_type(v_buf), v_buf, v_sl, v_sc))
}

func fn_read_triple_string(args ...any) any {
	v_s := args[0]
	v_sl := args[1]
	v_sc := args[2]
	v_buf := args[3]
	if isTruthy(fn_s_eof(v_s)) {
		return fn_s_emit(v_s, fn_Token("STRING", v_buf, v_sl, v_sc))
	}
	v_p := fn_s_pos(v_s)
	v_ln := fn_s_len(v_s)
	if isTruthy(vLt(vAdd(v_p, int64(2)), v_ln)) {
		if isTruthy(vEq(fn_s_ch(v_s), "\"")) {
			if isTruthy(vEq(fn_s_at(v_s, vAdd(v_p, int64(1))), "\"")) {
				if isTruthy(vEq(fn_s_at(v_s, vAdd(v_p, int64(2))), "\"")) {
					v_s2 := fn_s_advance(fn_s_advance(fn_s_advance(v_s)))
					return fn_s_emit(v_s2, fn_Token("STRING", v_buf, v_sl, v_sc))
				}
			}
		}
	}
	return fn_read_triple_string(fn_s_advance(v_s), v_sl, v_sc, vAdd(v_buf, fn_s_ch(v_s)))
}

func fn_read_string_body(args ...any) any {
	v_s := args[0]
	v_buf := args[1]
	if isTruthy(fn_s_eof(v_s)) {
		return map[string]any{"text": v_buf, "is_interp": false, "state": v_s}
	}
	v_ch := fn_s_ch(v_s)
	if isTruthy(vEq(v_ch, "\"")) {
		return map[string]any{"text": v_buf, "is_interp": false, "state": v_s}
	}
	if isTruthy(vEq(v_ch, "\\")) {
		v_p := vAdd(fn_s_pos(v_s), int64(1))
		if isTruthy(vLt(v_p, fn_s_len(v_s))) {
			v_next := fn_s_at(v_s, v_p)
			if isTruthy(vEq(v_next, "$")) {
				return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(v_buf, "$"))
			}
			if isTruthy(vEq(v_next, "n")) {
				return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(v_buf, `
`))
			}
			if isTruthy(vEq(v_next, "t")) {
				return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(v_buf, "\t"))
			}
			if isTruthy(vEq(v_next, "\\")) {
				return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(v_buf, "\\"))
			}
			if isTruthy(vEq(v_next, "\"")) {
				return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(v_buf, "\""))
			}
			return fn_read_string_body(fn_s_advance(fn_s_advance(v_s)), vAdd(vAdd(v_buf, "\\"), v_next))
		}
	}
	if isTruthy(vEq(v_ch, "$")) {
		v_p := vAdd(fn_s_pos(v_s), int64(1))
		if isTruthy(vLt(v_p, fn_s_len(v_s))) {
			if isTruthy(vEq(fn_s_at(v_s, v_p), "{")) {
				return map[string]any{"text": v_buf, "is_interp": true, "state": v_s}
			}
		}
	}
	return fn_read_string_body(fn_s_advance(v_s), vAdd(v_buf, v_ch))
}

func fn_read_string(args ...any) any {
	v_s := args[0]
	v_sl := args[1]
	v_sc := args[2]
	v_s2 := fn_s_advance(v_s)
	v_result := fn_read_string_body(v_s2, "")
	v_text := b_get(v_result, "text")
	v_is_interp := b_get(v_result, "is_interp")
	v_s3 := b_get(v_result, "state")
	if isTruthy(vEq(v_is_interp, false)) {
		v_s4 := fn_s_advance(v_s3)
		return fn_s_emit(v_s4, fn_Token("STRING", v_text, v_sl, v_sc))
	}
	v_s4 := fn_s_emit(v_s3, fn_Token("STRING", v_text, v_sl, v_sc))
	v_s5 := fn_s_advance(fn_s_advance(v_s4))
	v_new_depth := vAdd(b_get(v_s5, "interp_depth"), int64(1))
	v_s6 := b_set(v_s5, "interp_depth", v_new_depth)
	return fn_s_emit(v_s6, fn_Token("INTERP_START", "${", fn_s_line(v_s6), fn_s_col(v_s6)))
}

func fn_read_string_continuation(args ...any) any {
	v_s := args[0]
	v_sl := args[1]
	v_sc := args[2]
	v_s2 := fn_s_emit(v_s, fn_Token("INTERP_END", "}", v_sl, v_sc))
	v_result := fn_read_string_body(v_s2, "")
	v_text := b_get(v_result, "text")
	v_is_interp := b_get(v_result, "is_interp")
	v_s3 := b_get(v_result, "state")
	if isTruthy(vEq(v_is_interp, false)) {
		v_s4 := fn_s_advance(v_s3)
		return fn_s_emit(v_s4, fn_Token("STRING", v_text, fn_s_line(v_s4), fn_s_col(v_s4)))
	}
	v_s4 := fn_s_emit(v_s3, fn_Token("STRING", v_text, fn_s_line(v_s3), fn_s_col(v_s3)))
	v_s5 := fn_s_advance(fn_s_advance(v_s4))
	v_new_depth := vAdd(b_get(v_s5, "interp_depth"), int64(1))
	v_s6 := b_set(v_s5, "interp_depth", v_new_depth)
	return fn_s_emit(v_s6, fn_Token("INTERP_START", "${", fn_s_line(v_s6), fn_s_col(v_s6)))
}

func fn_scan_token(args ...any) any {
	v_s := args[0]
	v_s2 := fn_skip_whitespace(v_s)
	if isTruthy(fn_s_eof(v_s2)) {
		return fn_s_emit(v_s2, fn_Token("EOF", "", fn_s_line(v_s2), fn_s_col(v_s2)))
	}
	v_ch := fn_s_ch(v_s2)
	v_sl := fn_s_line(v_s2)
	v_sc := fn_s_col(v_s2)
	if isTruthy(vEq(v_ch, "/")) {
		if isTruthy(vEq(fn_s_peek(v_s2), "/")) {
			return fn_scan_token(fn_skip_line_comment(fn_s_advance(fn_s_advance(v_s2))))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("SLASH", "/", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "\"")) {
		v_p := fn_s_pos(v_s2)
		if isTruthy(vLt(vAdd(v_p, int64(2)), fn_s_len(v_s2))) {
			if isTruthy(vEq(fn_s_at(v_s2, vAdd(v_p, int64(1))), "\"")) {
				if isTruthy(vEq(fn_s_at(v_s2, vAdd(v_p, int64(2))), "\"")) {
					v_s3 := fn_s_advance(fn_s_advance(fn_s_advance(v_s2)))
					if isTruthy(vEq(fn_s_eof(v_s3), false)) {
						if isTruthy(vEq(fn_s_ch(v_s3), `
`)) {
							return fn_read_triple_string(fn_s_advance(v_s3), v_sl, v_sc, "")
						}
					}
					return fn_read_triple_string(v_s3, v_sl, v_sc, "")
				}
			}
		}
		return fn_read_string(v_s2, v_sl, v_sc)
	}
	if isTruthy(vEq(v_ch, "|")) {
		if isTruthy(vEq(fn_s_peek(v_s2), ">")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("PIPE", "|>", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("ILLEGAL", "|", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "=")) {
		v_next := fn_s_peek(v_s2)
		if isTruthy(vEq(v_next, ">")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("ARROW", "=>", v_sl, v_sc))
		}
		if isTruthy(vEq(v_next, "=")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("EQ", "==", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("EQUAL", "=", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "!")) {
		if isTruthy(vEq(fn_s_peek(v_s2), "=")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("NEQ", "!=", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("ILLEGAL", "!", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "<")) {
		if isTruthy(vEq(fn_s_peek(v_s2), "=")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("LTE", "<=", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("LT", "<", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, ">")) {
		if isTruthy(vEq(fn_s_peek(v_s2), "=")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("GTE", ">=", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("GT", ">", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "(")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("LPAREN", "(", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, ")")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("RPAREN", ")", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "{")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("LBRACE", "{", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "}")) {
		if isTruthy(vGt(b_get(v_s2, "interp_depth"), int64(0))) {
			v_s3 := b_set(v_s2, "interp_depth", vSub(b_get(v_s2, "interp_depth"), int64(1)))
			return fn_read_string_continuation(fn_s_advance(v_s3), v_sl, v_sc)
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("RBRACE", "}", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "[")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("LBRACKET", "[", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "]")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("RBRACKET", "]", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, ",")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("COMMA", ",", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, ":")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("COLON", ":", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, ".")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("DOT", ".", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "+")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("PLUS", "+", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "-")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("MINUS", "-", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "*")) {
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("STAR", "*", v_sl, v_sc))
	}
	if isTruthy(vEq(v_ch, "#")) {
		if isTruthy(vEq(fn_s_peek(v_s2), "{")) {
			return fn_s_emit(fn_s_advance(fn_s_advance(v_s2)), fn_Token("HASH_LBRACE", "#{", v_sl, v_sc))
		}
		return fn_s_emit(fn_s_advance(v_s2), fn_Token("ILLEGAL", "#", v_sl, v_sc))
	}
	if isTruthy(fn_is_digit(v_ch)) {
		return fn_read_int(fn_s_advance(v_s2), v_sl, v_sc, v_ch)
	}
	if isTruthy(fn_is_alpha(v_ch)) {
		return fn_read_ident(fn_s_advance(v_s2), v_sl, v_sc, v_ch)
	}
	return fn_s_emit(fn_s_advance(v_s2), fn_Token("ILLEGAL", v_ch, v_sl, v_sc))
}

func fn_tokenize(args ...any) any {
	v_input := args[0]
	var v_s any = map[string]any{"input": v_input, "len": b_length(v_input), "pos": int64(0), "line": int64(1), "col": int64(1), "interp_depth": int64(0), "tokens": []any{}}
	var v_done any = false
	for isTruthy(vEq(v_done, false)) {
		v_s = fn_scan_token(v_s)
		v_toks := b_get(v_s, "tokens")
		v_last := b_index(v_toks, vSub(b_length(v_toks), int64(1)))
		if isTruthy(vEq(b_get(v_last, "type"), "EOF")) {
			v_done = true
		}
	}
	return b_get(v_s, "tokens")
}

func fn_Program(args ...any) any {
	return map[string]any{"__type": "Program", "decls": args[0]}
}

func fn_FnDecl(args ...any) any {
	return map[string]any{"__type": "FnDecl", "name": args[0], "params": args[1], "return_type": args[2], "body": args[3]}
}

func fn_Param(args ...any) any {
	return map[string]any{"__type": "Param", "name": args[0], "type_name": args[1]}
}

func fn_LetStmt(args ...any) any {
	return map[string]any{"__type": "LetStmt", "name": args[0], "value": args[1], "mutable": args[2]}
}

func fn_IfExpr(args ...any) any {
	return map[string]any{"__type": "IfExpr", "condition": args[0], "then_body": args[1], "else_body": args[2]}
}

func fn_MatchExpr(args ...any) any {
	return map[string]any{"__type": "MatchExpr", "subject": args[0], "arms": args[1]}
}

func fn_MatchArm(args ...any) any {
	return map[string]any{"__type": "MatchArm", "pattern": args[0], "body": args[1]}
}

func fn_BinaryExpr(args ...any) any {
	return map[string]any{"__type": "BinaryExpr", "op": args[0], "left": args[1], "right": args[2]}
}

func fn_CallExpr(args ...any) any {
	return map[string]any{"__type": "CallExpr", "fn_node": args[0], "args": args[1]}
}

func fn_PipeExpr(args ...any) any {
	return map[string]any{"__type": "PipeExpr", "left": args[0], "right": args[1]}
}

func fn_ArrayLit(args ...any) any {
	return map[string]any{"__type": "ArrayLit", "elements": args[0]}
}

func fn_HashLit(args ...any) any {
	return map[string]any{"__type": "HashLit", "pairs": args[0]}
}

func fn_HashPair(args ...any) any {
	return map[string]any{"__type": "HashPair", "key": args[0], "value": args[1]}
}

func fn_DotExpr(args ...any) any {
	return map[string]any{"__type": "DotExpr", "object": args[0], "field": args[1]}
}

func fn_FnLit(args ...any) any {
	return map[string]any{"__type": "FnLit", "params": args[0], "body": args[1]}
}

func fn_FanOutExpr(args ...any) any {
	return map[string]any{"__type": "FanOutExpr", "value": args[0], "fns": args[1]}
}

func fn_IntLit(args ...any) any {
	return map[string]any{"__type": "IntLit", "value": args[0]}
}

func fn_StringLit(args ...any) any {
	return map[string]any{"__type": "StringLit", "value": args[0]}
}

func fn_BoolLit(args ...any) any {
	return map[string]any{"__type": "BoolLit", "value": args[0]}
}

func fn_Ident(args ...any) any {
	return map[string]any{"__type": "Ident", "name": args[0]}
}

func fn_ImportDecl(args ...any) any {
	return map[string]any{"__type": "ImportDecl", "path": args[0]}
}

func fn_WhileExpr(args ...any) any {
	return map[string]any{"__type": "WhileExpr", "condition": args[0], "body": args[1]}
}

func fn_AssignStmt(args ...any) any {
	return map[string]any{"__type": "AssignStmt", "name": args[0], "value": args[1]}
}

func fn_StructDecl(args ...any) any {
	return map[string]any{"__type": "StructDecl", "name": args[0], "fields": args[1]}
}

func fn_TypePattern(args ...any) any {
	return map[string]any{"__type": "TypePattern", "type_name": args[0], "binding": args[1]}
}

func fn_ReturnStmt(args ...any) any {
	return map[string]any{"__type": "ReturnStmt", "value": args[0]}
}

func fn_InterpLit(args ...any) any {
	return map[string]any{"__type": "InterpLit", "parts": args[0]}
}

func fn_make_ok(args ...any) any {
	v_node := args[0]
	return map[string]any{"node": v_node, "error": ""}
}

func fn_make_err(args ...any) any {
	v_msg := args[0]
	return map[string]any{"node": int64(0), "error": v_msg}
}

func fn_is_err(args ...any) any {
	v_r := args[0]
	return vNeq(b_get(v_r, "error"), "")
}

func fn_p_current(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	return b_index(v_tokens, v_pos)
}

func fn_p_check(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_typ := args[2]
	return vEq(b_get(fn_p_current(v_tokens, v_pos), "type"), v_typ)
}

func fn_p_at_end(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	return fn_p_check(v_tokens, v_pos, "EOF")
}

func fn_p_peek_type(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	if isTruthy(vLt(vAdd(v_pos, int64(1)), b_length(v_tokens))) {
		return b_get(b_index(v_tokens, vAdd(v_pos, int64(1))), "type")
	}
	return "EOF"
}

func fn_parse(args ...any) any {
	v_tokens := args[0]
	var v_pos any = int64(0)
	var v_decls any = []any{}
	for isTruthy(vEq(fn_p_at_end(v_tokens, v_pos), false)) {
		v_result := fn_parse_decl(v_tokens, v_pos)
		if isTruthy(fn_is_err(v_result)) {
			return v_result
		}
		v_pos = b_get(v_result, "pos")
		v_decls = b_push(v_decls, b_get(v_result, "node"))
	}
	return fn_make_ok(fn_Program(v_decls))
}

func fn_parse_decl(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_tok := fn_p_current(v_tokens, v_pos)
	if isTruthy(vEq(b_get(v_tok, "type"), "IMPORT")) {
		return fn_parse_import(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "FN")) {
		return fn_parse_fn_decl(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "STRUCT")) {
		return fn_parse_struct_decl(v_tokens, v_pos)
	}
	return fn_make_err(vAdd(vAdd(vAdd(vAdd(vAdd("expected declaration, got ", b_get(v_tok, "type")), "("), b_get(v_tok, "literal")), ") at line "), b_to_string(b_get(v_tok, "line"))))
}

func fn_parse_import(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_tok := fn_p_current(v_tokens, v_p)
	if isTruthy(vNeq(b_get(v_tok, "type"), "STRING")) {
		return fn_make_err(vAdd("expected string after 'import', got ", b_get(v_tok, "type")))
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_ImportDecl(b_get(v_tok, "literal")), "error": "", "pos": v_p}
}

func fn_parse_fn_decl(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_name_tok := fn_p_current(v_tokens, v_p)
	if isTruthy(vNeq(b_get(v_name_tok, "type"), "IDENT")) {
		return fn_make_err(vAdd("expected function name, got ", b_get(v_name_tok, "type")))
	}
	v_name := b_get(v_name_tok, "literal")
	v_p = vAdd(v_p, int64(1))
	v_params_result := fn_parse_param_list(v_tokens, v_p)
	if isTruthy(fn_is_err(v_params_result)) {
		return v_params_result
	}
	v_p = b_get(v_params_result, "pos")
	v_params := b_get(v_params_result, "node")
	var v_return_type any = ""
	if isTruthy(fn_p_check(v_tokens, v_p, "COLON")) {
		v_p = vAdd(v_p, int64(1))
		v_rt := fn_p_current(v_tokens, v_p)
		if isTruthy(vNeq(b_get(v_rt, "type"), "IDENT")) {
			return fn_make_err(vAdd("expected return type after ':', got ", b_get(v_rt, "type")))
		}
		v_return_type = b_get(v_rt, "literal")
		v_p = vAdd(v_p, int64(1))
	}
	v_body_result := fn_parse_block(v_tokens, v_p)
	if isTruthy(fn_is_err(v_body_result)) {
		return v_body_result
	}
	v_p = b_get(v_body_result, "pos")
	v_body := b_get(v_body_result, "node")
	return map[string]any{"node": fn_FnDecl(v_name, v_params, v_return_type, v_body), "error": "", "pos": v_p}
}

func fn_parse_param_list(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	if isTruthy(vEq(fn_p_check(v_tokens, v_pos, "LPAREN"), false)) {
		return fn_make_err("expected '(' for parameter list")
	}
	var v_p any = vAdd(v_pos, int64(1))
	var v_params any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RPAREN"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in parameter list")
		}
		v_name_tok := fn_p_current(v_tokens, v_p)
		if isTruthy(vNeq(b_get(v_name_tok, "type"), "IDENT")) {
			return fn_make_err(vAdd("expected parameter name, got ", b_get(v_name_tok, "type")))
		}
		v_p = vAdd(v_p, int64(1))
		var v_type_name any = ""
		if isTruthy(fn_p_check(v_tokens, v_p, "COLON")) {
			v_p = vAdd(v_p, int64(1))
			v_tt := fn_p_current(v_tokens, v_p)
			if isTruthy(vNeq(b_get(v_tt, "type"), "IDENT")) {
				return fn_make_err(vAdd("expected type after ':', got ", b_get(v_tt, "type")))
			}
			v_type_name = b_get(v_tt, "literal")
			v_p = vAdd(v_p, int64(1))
		}
		v_params = b_push(v_params, fn_Param(b_get(v_name_tok, "literal"), v_type_name))
		if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
			v_p = vAdd(v_p, int64(1))
		}
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": v_params, "error": "", "pos": v_p}
}

func fn_parse_block(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	if isTruthy(vEq(fn_p_check(v_tokens, v_pos, "LBRACE"), false)) {
		return fn_make_err(vAdd("expected '{', got ", b_get(fn_p_current(v_tokens, v_pos), "type")))
	}
	var v_p any = vAdd(v_pos, int64(1))
	var v_stmts any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACE"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in block")
		}
		v_result := fn_parse_stmt(v_tokens, v_p)
		if isTruthy(fn_is_err(v_result)) {
			return v_result
		}
		v_p = b_get(v_result, "pos")
		v_stmts = b_push(v_stmts, b_get(v_result, "node"))
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": v_stmts, "error": "", "pos": v_p}
}

func fn_parse_stmt(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_tok := fn_p_current(v_tokens, v_pos)
	if isTruthy(vEq(b_get(v_tok, "type"), "LET")) {
		return fn_parse_let(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "WHILE")) {
		return fn_parse_while(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "RETURN")) {
		return fn_parse_return(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "IDENT")) {
		if isTruthy(vEq(fn_p_peek_type(v_tokens, v_pos), "EQUAL")) {
			v_name := b_get(v_tok, "literal")
			var v_p any = vAdd(v_pos, int64(2))
			v_val_result := fn_parse_expr(v_tokens, v_p)
			if isTruthy(fn_is_err(v_val_result)) {
				return v_val_result
			}
			v_p = b_get(v_val_result, "pos")
			return map[string]any{"node": fn_AssignStmt(v_name, b_get(v_val_result, "node")), "error": "", "pos": v_p}
		}
	}
	return fn_parse_expr(v_tokens, v_pos)
}

func fn_parse_let(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	var v_mutable any = false
	if isTruthy(fn_p_check(v_tokens, v_p, "MUT")) {
		v_mutable = true
		v_p = vAdd(v_p, int64(1))
	}
	v_name_tok := fn_p_current(v_tokens, v_p)
	if isTruthy(vNeq(b_get(v_name_tok, "type"), "IDENT")) {
		return fn_make_err(vAdd("expected variable name after 'let', got ", b_get(v_name_tok, "type")))
	}
	v_p = vAdd(v_p, int64(1))
	if isTruthy(vEq(fn_p_check(v_tokens, v_p, "EQUAL"), false)) {
		return fn_make_err("expected '=' after variable name")
	}
	v_p = vAdd(v_p, int64(1))
	v_val_result := fn_parse_expr(v_tokens, v_p)
	if isTruthy(fn_is_err(v_val_result)) {
		return v_val_result
	}
	v_p = b_get(v_val_result, "pos")
	return map[string]any{"node": fn_LetStmt(b_get(v_name_tok, "literal"), b_get(v_val_result, "node"), v_mutable), "error": "", "pos": v_p}
}

func fn_parse_while(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_cond_result := fn_parse_expr(v_tokens, v_p)
	if isTruthy(fn_is_err(v_cond_result)) {
		return v_cond_result
	}
	v_p = b_get(v_cond_result, "pos")
	v_body_result := fn_parse_block(v_tokens, v_p)
	if isTruthy(fn_is_err(v_body_result)) {
		return v_body_result
	}
	v_p = b_get(v_body_result, "pos")
	return map[string]any{"node": fn_WhileExpr(b_get(v_cond_result, "node"), b_get(v_body_result, "node")), "error": "", "pos": v_p}
}

func fn_parse_return(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	if isTruthy(fn_p_check(v_tokens, v_p, "RBRACE")) {
		return map[string]any{"node": fn_ReturnStmt(int64(0)), "error": "", "pos": v_p}
	}
	if isTruthy(fn_p_at_end(v_tokens, v_p)) {
		return map[string]any{"node": fn_ReturnStmt(int64(0)), "error": "", "pos": v_p}
	}
	v_val_result := fn_parse_expr(v_tokens, v_p)
	if isTruthy(fn_is_err(v_val_result)) {
		return v_val_result
	}
	v_p = b_get(v_val_result, "pos")
	return map[string]any{"node": fn_ReturnStmt(b_get(v_val_result, "node")), "error": "", "pos": v_p}
}

func fn_parse_expr(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	return fn_parse_pipe(v_tokens, v_pos)
}

func fn_parse_pipe(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_left_result := fn_parse_comparison(v_tokens, v_pos)
	if isTruthy(fn_is_err(v_left_result)) {
		return v_left_result
	}
	var v_p any = b_get(v_left_result, "pos")
	var v_left any = b_get(v_left_result, "node")
	for isTruthy(fn_p_check(v_tokens, v_p, "PIPE")) {
		v_p = vAdd(v_p, int64(1))
		if isTruthy(fn_p_check(v_tokens, v_p, "LBRACKET")) {
			v_p = vAdd(v_p, int64(1))
			var v_fns any = []any{}
			for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACKET"), false)) {
				v_fn_result := fn_parse_expr(v_tokens, v_p)
				if isTruthy(fn_is_err(v_fn_result)) {
					return v_fn_result
				}
				v_p = b_get(v_fn_result, "pos")
				v_fns = b_push(v_fns, b_get(v_fn_result, "node"))
				if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
					v_p = vAdd(v_p, int64(1))
				}
			}
			v_p = vAdd(v_p, int64(1))
			v_left = fn_FanOutExpr(v_left, v_fns)
		} else {
			if isTruthy(fn_p_check(v_tokens, v_p, "MATCH")) {
				v_p = vAdd(v_p, int64(1))
				v_match_result := fn_parse_match_body(v_tokens, v_p, v_left)
				if isTruthy(fn_is_err(v_match_result)) {
					return v_match_result
				}
				v_p = b_get(v_match_result, "pos")
				v_left = b_get(v_match_result, "node")
			} else {
				v_right_result := fn_parse_comparison(v_tokens, v_p)
				if isTruthy(fn_is_err(v_right_result)) {
					return v_right_result
				}
				v_p = b_get(v_right_result, "pos")
				v_right := b_get(v_right_result, "node")
				if isTruthy(vEq(b_type_of(v_right), "map")) {
					if isTruthy(b_has_key(v_right, "__type")) {
						if isTruthy(vEq(b_get(v_right, "__type"), "CallExpr")) {
							v_existing_args := b_get(v_right, "args")
							v_new_args := b_concat([]any{v_left}, v_existing_args)
							v_left = fn_CallExpr(b_get(v_right, "fn_node"), v_new_args)
						} else {
							v_left = fn_CallExpr(v_right, []any{v_left})
						}
					} else {
						v_left = fn_CallExpr(v_right, []any{v_left})
					}
				} else {
					v_left = fn_CallExpr(v_right, []any{v_left})
				}
			}
		}
	}
	return map[string]any{"node": v_left, "error": "", "pos": v_p}
}

func fn_is_comparison_type(args ...any) any {
	v_typ := args[0]
	if isTruthy(vEq(v_typ, "EQ")) {
		return true
	}
	if isTruthy(vEq(v_typ, "NEQ")) {
		return true
	}
	if isTruthy(vEq(v_typ, "LT")) {
		return true
	}
	if isTruthy(vEq(v_typ, "GT")) {
		return true
	}
	if isTruthy(vEq(v_typ, "LTE")) {
		return true
	}
	if isTruthy(vEq(v_typ, "GTE")) {
		return true
	}
	return false
}

func fn_parse_comparison(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_left_result := fn_parse_add_sub(v_tokens, v_pos)
	if isTruthy(fn_is_err(v_left_result)) {
		return v_left_result
	}
	var v_p any = b_get(v_left_result, "pos")
	var v_left any = b_get(v_left_result, "node")
	for isTruthy(fn_is_comparison_type(b_get(fn_p_current(v_tokens, v_p), "type"))) {
		v_op := b_get(fn_p_current(v_tokens, v_p), "literal")
		v_p = vAdd(v_p, int64(1))
		v_right_result := fn_parse_add_sub(v_tokens, v_p)
		if isTruthy(fn_is_err(v_right_result)) {
			return v_right_result
		}
		v_p = b_get(v_right_result, "pos")
		v_left = fn_BinaryExpr(v_op, v_left, b_get(v_right_result, "node"))
	}
	return map[string]any{"node": v_left, "error": "", "pos": v_p}
}

func fn_is_add_sub(args ...any) any {
	v_typ := args[0]
	if isTruthy(vEq(v_typ, "PLUS")) {
		return true
	}
	if isTruthy(vEq(v_typ, "MINUS")) {
		return true
	}
	return false
}

func fn_parse_add_sub(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_left_result := fn_parse_mul_div(v_tokens, v_pos)
	if isTruthy(fn_is_err(v_left_result)) {
		return v_left_result
	}
	var v_p any = b_get(v_left_result, "pos")
	var v_left any = b_get(v_left_result, "node")
	for isTruthy(fn_is_add_sub(b_get(fn_p_current(v_tokens, v_p), "type"))) {
		v_op := b_get(fn_p_current(v_tokens, v_p), "literal")
		v_p = vAdd(v_p, int64(1))
		v_right_result := fn_parse_mul_div(v_tokens, v_p)
		if isTruthy(fn_is_err(v_right_result)) {
			return v_right_result
		}
		v_p = b_get(v_right_result, "pos")
		v_left = fn_BinaryExpr(v_op, v_left, b_get(v_right_result, "node"))
	}
	return map[string]any{"node": v_left, "error": "", "pos": v_p}
}

func fn_is_mul_div(args ...any) any {
	v_typ := args[0]
	if isTruthy(vEq(v_typ, "STAR")) {
		return true
	}
	if isTruthy(vEq(v_typ, "SLASH")) {
		return true
	}
	return false
}

func fn_parse_mul_div(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_left_result := fn_parse_postfix(v_tokens, v_pos)
	if isTruthy(fn_is_err(v_left_result)) {
		return v_left_result
	}
	var v_p any = b_get(v_left_result, "pos")
	var v_left any = b_get(v_left_result, "node")
	for isTruthy(fn_is_mul_div(b_get(fn_p_current(v_tokens, v_p), "type"))) {
		v_op := b_get(fn_p_current(v_tokens, v_p), "literal")
		v_p = vAdd(v_p, int64(1))
		v_right_result := fn_parse_postfix(v_tokens, v_p)
		if isTruthy(fn_is_err(v_right_result)) {
			return v_right_result
		}
		v_p = b_get(v_right_result, "pos")
		v_left = fn_BinaryExpr(v_op, v_left, b_get(v_right_result, "node"))
	}
	return map[string]any{"node": v_left, "error": "", "pos": v_p}
}

func fn_parse_postfix(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_left_result := fn_parse_primary(v_tokens, v_pos)
	if isTruthy(fn_is_err(v_left_result)) {
		return v_left_result
	}
	var v_p any = b_get(v_left_result, "pos")
	var v_left any = b_get(v_left_result, "node")
	var v_cont any = true
	for isTruthy(v_cont) {
		if isTruthy(fn_p_check(v_tokens, v_p, "LPAREN")) {
			v_call_result := fn_parse_call_args(v_tokens, v_p, v_left)
			if isTruthy(fn_is_err(v_call_result)) {
				return v_call_result
			}
			v_p = b_get(v_call_result, "pos")
			v_left = b_get(v_call_result, "node")
		} else {
			if isTruthy(fn_p_check(v_tokens, v_p, "DOT")) {
				v_p = vAdd(v_p, int64(1))
				v_field_tok := fn_p_current(v_tokens, v_p)
				if isTruthy(vNeq(b_get(v_field_tok, "type"), "IDENT")) {
					return fn_make_err(vAdd("expected field name after '.', got ", b_get(v_field_tok, "type")))
				}
				v_p = vAdd(v_p, int64(1))
				v_left = fn_DotExpr(v_left, b_get(v_field_tok, "literal"))
			} else {
				v_cont = false
			}
		}
	}
	return map[string]any{"node": v_left, "error": "", "pos": v_p}
}

func fn_parse_call_args(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_fn_node := args[2]
	var v_p any = vAdd(v_pos, int64(1))
	var v_args any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RPAREN"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in call arguments")
		}
		v_arg_result := fn_parse_expr(v_tokens, v_p)
		if isTruthy(fn_is_err(v_arg_result)) {
			return v_arg_result
		}
		v_p = b_get(v_arg_result, "pos")
		v_args = b_push(v_args, b_get(v_arg_result, "node"))
		if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
			v_p = vAdd(v_p, int64(1))
		}
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_CallExpr(v_fn_node, v_args), "error": "", "pos": v_p}
}

func fn_parse_primary(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_tok := fn_p_current(v_tokens, v_pos)
	if isTruthy(vEq(b_get(v_tok, "type"), "INT")) {
		return map[string]any{"node": fn_IntLit(b_to_int(b_get(v_tok, "literal"))), "error": "", "pos": vAdd(v_pos, int64(1))}
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "STRING")) {
		if isTruthy(vEq(fn_p_peek_type(v_tokens, v_pos), "INTERP_START")) {
			return fn_parse_interp_lit(v_tokens, v_pos)
		}
		return map[string]any{"node": fn_StringLit(b_get(v_tok, "literal")), "error": "", "pos": vAdd(v_pos, int64(1))}
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "TRUE")) {
		return map[string]any{"node": fn_BoolLit(true), "error": "", "pos": vAdd(v_pos, int64(1))}
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "FALSE")) {
		return map[string]any{"node": fn_BoolLit(false), "error": "", "pos": vAdd(v_pos, int64(1))}
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "IDENT")) {
		return map[string]any{"node": fn_Ident(b_get(v_tok, "literal")), "error": "", "pos": vAdd(v_pos, int64(1))}
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "FN")) {
		return fn_parse_fn_lit(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "LBRACKET")) {
		return fn_parse_array_lit(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "HASH_LBRACE")) {
		return fn_parse_hash_lit(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "IF")) {
		return fn_parse_if_expr(v_tokens, v_pos)
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "MATCH")) {
		return fn_make_err("match must be used with pipe: expr |> match { ... }")
	}
	if isTruthy(vEq(b_get(v_tok, "type"), "LPAREN")) {
		var v_p any = vAdd(v_pos, int64(1))
		v_inner_result := fn_parse_expr(v_tokens, v_p)
		if isTruthy(fn_is_err(v_inner_result)) {
			return v_inner_result
		}
		v_p = b_get(v_inner_result, "pos")
		if isTruthy(vEq(fn_p_check(v_tokens, v_p, "RPAREN"), false)) {
			return fn_make_err("expected ')' after expression")
		}
		return map[string]any{"node": b_get(v_inner_result, "node"), "error": "", "pos": vAdd(v_p, int64(1))}
	}
	return fn_make_err(vAdd(vAdd(vAdd(vAdd(vAdd("unexpected token: ", b_get(v_tok, "type")), "("), b_get(v_tok, "literal")), ") at line "), b_to_string(b_get(v_tok, "line"))))
}

func fn_parse_fn_lit(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_params_result := fn_parse_param_list(v_tokens, v_p)
	if isTruthy(fn_is_err(v_params_result)) {
		return v_params_result
	}
	v_p = b_get(v_params_result, "pos")
	v_body_result := fn_parse_block(v_tokens, v_p)
	if isTruthy(fn_is_err(v_body_result)) {
		return v_body_result
	}
	v_p = b_get(v_body_result, "pos")
	return map[string]any{"node": fn_FnLit(b_get(v_params_result, "node"), b_get(v_body_result, "node")), "error": "", "pos": v_p}
}

func fn_parse_array_lit(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	var v_elems any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACKET"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in array literal")
		}
		v_elem_result := fn_parse_expr(v_tokens, v_p)
		if isTruthy(fn_is_err(v_elem_result)) {
			return v_elem_result
		}
		v_p = b_get(v_elem_result, "pos")
		v_elems = b_push(v_elems, b_get(v_elem_result, "node"))
		if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
			v_p = vAdd(v_p, int64(1))
		}
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_ArrayLit(v_elems), "error": "", "pos": v_p}
}

func fn_parse_hash_lit(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	var v_pairs any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACE"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in hash literal")
		}
		v_key_tok := fn_p_current(v_tokens, v_p)
		if isTruthy(vNeq(b_get(v_key_tok, "type"), "IDENT")) {
			return fn_make_err(vAdd("expected key name in hash, got ", b_get(v_key_tok, "type")))
		}
		v_p = vAdd(v_p, int64(1))
		if isTruthy(vEq(fn_p_check(v_tokens, v_p, "COLON"), false)) {
			return fn_make_err("expected ':' after hash key")
		}
		v_p = vAdd(v_p, int64(1))
		v_val_result := fn_parse_expr(v_tokens, v_p)
		if isTruthy(fn_is_err(v_val_result)) {
			return v_val_result
		}
		v_p = b_get(v_val_result, "pos")
		v_pairs = b_push(v_pairs, fn_HashPair(b_get(v_key_tok, "literal"), b_get(v_val_result, "node")))
		if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
			v_p = vAdd(v_p, int64(1))
		}
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_HashLit(v_pairs), "error": "", "pos": v_p}
}

func fn_parse_if_expr(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_cond_result := fn_parse_expr(v_tokens, v_p)
	if isTruthy(fn_is_err(v_cond_result)) {
		return v_cond_result
	}
	v_p = b_get(v_cond_result, "pos")
	v_then_result := fn_parse_block(v_tokens, v_p)
	if isTruthy(fn_is_err(v_then_result)) {
		return v_then_result
	}
	v_p = b_get(v_then_result, "pos")
	var v_else_body any = []any{}
	if isTruthy(fn_p_check(v_tokens, v_p, "ELSE")) {
		v_p = vAdd(v_p, int64(1))
		if isTruthy(fn_p_check(v_tokens, v_p, "IF")) {
			v_elif_result := fn_parse_if_expr(v_tokens, v_p)
			if isTruthy(fn_is_err(v_elif_result)) {
				return v_elif_result
			}
			v_p = b_get(v_elif_result, "pos")
			v_else_body = []any{b_get(v_elif_result, "node")}
		} else {
			v_else_result := fn_parse_block(v_tokens, v_p)
			if isTruthy(fn_is_err(v_else_result)) {
				return v_else_result
			}
			v_p = b_get(v_else_result, "pos")
			v_else_body = b_get(v_else_result, "node")
		}
	}
	return map[string]any{"node": fn_IfExpr(b_get(v_cond_result, "node"), b_get(v_then_result, "node"), v_else_body), "error": "", "pos": v_p}
}

func fn_parse_match_body(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_subject := args[2]
	if isTruthy(vEq(fn_p_check(v_tokens, v_pos, "LBRACE"), false)) {
		return fn_make_err("expected '{' after 'match'")
	}
	var v_p any = vAdd(v_pos, int64(1))
	var v_arms any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACE"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in match expression")
		}
		v_pattern_result := fn_parse_match_pattern(v_tokens, v_p)
		if isTruthy(fn_is_err(v_pattern_result)) {
			return v_pattern_result
		}
		v_p = b_get(v_pattern_result, "pos")
		if isTruthy(vEq(fn_p_check(v_tokens, v_p, "ARROW"), false)) {
			return fn_make_err(vAdd("expected '=>' in match arm, got ", b_get(fn_p_current(v_tokens, v_p), "type")))
		}
		v_p = vAdd(v_p, int64(1))
		v_body_result := fn_parse_expr(v_tokens, v_p)
		if isTruthy(fn_is_err(v_body_result)) {
			return v_body_result
		}
		v_p = b_get(v_body_result, "pos")
		v_arms = b_push(v_arms, fn_MatchArm(b_get(v_pattern_result, "node"), b_get(v_body_result, "node")))
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_MatchExpr(v_subject, v_arms), "error": "", "pos": v_p}
}

func fn_parse_match_pattern(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	v_tok := fn_p_current(v_tokens, v_pos)
	if isTruthy(vEq(b_get(v_tok, "type"), "IDENT")) {
		if isTruthy(vEq(fn_p_peek_type(v_tokens, v_pos), "LPAREN")) {
			v_type_name := b_get(v_tok, "literal")
			var v_p any = vAdd(v_pos, int64(2))
			v_bind_tok := fn_p_current(v_tokens, v_p)
			if isTruthy(vNeq(b_get(v_bind_tok, "type"), "IDENT")) {
				return fn_make_err(vAdd("expected binding name in type pattern, got ", b_get(v_bind_tok, "type")))
			}
			v_p = vAdd(v_p, int64(1))
			if isTruthy(vEq(fn_p_check(v_tokens, v_p, "RPAREN"), false)) {
				return fn_make_err("expected ')' after type pattern binding")
			}
			v_p = vAdd(v_p, int64(1))
			return map[string]any{"node": fn_TypePattern(v_type_name, b_get(v_bind_tok, "literal")), "error": "", "pos": v_p}
		}
	}
	return fn_parse_primary(v_tokens, v_pos)
}

func fn_parse_interp_lit(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = v_pos
	var v_parts any = []any{}
	v_first_tok := fn_p_current(v_tokens, v_p)
	v_parts = b_push(v_parts, fn_StringLit(b_get(v_first_tok, "literal")))
	v_p = vAdd(v_p, int64(1))
	var v_cont any = true
	for isTruthy(v_cont) {
		if isTruthy(fn_p_check(v_tokens, v_p, "INTERP_START")) {
			v_p = vAdd(v_p, int64(1))
			v_expr_result := fn_parse_expr(v_tokens, v_p)
			if isTruthy(fn_is_err(v_expr_result)) {
				return v_expr_result
			}
			v_p = b_get(v_expr_result, "pos")
			v_parts = b_push(v_parts, b_get(v_expr_result, "node"))
			if isTruthy(vEq(fn_p_check(v_tokens, v_p, "INTERP_END"), false)) {
				return fn_make_err("expected '}' to close interpolation")
			}
			v_p = vAdd(v_p, int64(1))
			if isTruthy(fn_p_check(v_tokens, v_p, "STRING")) {
				v_parts = b_push(v_parts, fn_StringLit(b_get(fn_p_current(v_tokens, v_p), "literal")))
				v_p = vAdd(v_p, int64(1))
			}
		} else {
			v_cont = false
		}
	}
	return map[string]any{"node": fn_InterpLit(v_parts), "error": "", "pos": v_p}
}

func fn_parse_struct_decl(args ...any) any {
	v_tokens := args[0]
	v_pos := args[1]
	var v_p any = vAdd(v_pos, int64(1))
	v_name_tok := fn_p_current(v_tokens, v_p)
	if isTruthy(vNeq(b_get(v_name_tok, "type"), "IDENT")) {
		return fn_make_err(vAdd("expected struct name, got ", b_get(v_name_tok, "type")))
	}
	v_p = vAdd(v_p, int64(1))
	if isTruthy(vEq(fn_p_check(v_tokens, v_p, "LBRACE"), false)) {
		return fn_make_err("expected '{' after struct name")
	}
	v_p = vAdd(v_p, int64(1))
	var v_fields any = []any{}
	for isTruthy(vEq(fn_p_check(v_tokens, v_p, "RBRACE"), false)) {
		if isTruthy(fn_p_at_end(v_tokens, v_p)) {
			return fn_make_err("unexpected EOF in struct declaration")
		}
		v_field_tok := fn_p_current(v_tokens, v_p)
		if isTruthy(vNeq(b_get(v_field_tok, "type"), "IDENT")) {
			return fn_make_err(vAdd("expected field name, got ", b_get(v_field_tok, "type")))
		}
		v_fields = b_push(v_fields, b_get(v_field_tok, "literal"))
		v_p = vAdd(v_p, int64(1))
		if isTruthy(fn_p_check(v_tokens, v_p, "COLON")) {
			v_p = vAdd(v_p, int64(1))
			if isTruthy(fn_p_check(v_tokens, v_p, "IDENT")) {
				v_p = vAdd(v_p, int64(1))
			}
		}
		if isTruthy(fn_p_check(v_tokens, v_p, "COMMA")) {
			v_p = vAdd(v_p, int64(1))
		}
	}
	v_p = vAdd(v_p, int64(1))
	return map[string]any{"node": fn_StructDecl(b_get(v_name_tok, "literal"), v_fields), "error": "", "pos": v_p}
}

func fn_is_builtin(args ...any) any {
	v_name := args[0]
	if valuesEqual(v_name, "print") {
		return true
	}
	if valuesEqual(v_name, "uppercase") {
		return true
	}
	if valuesEqual(v_name, "length") {
		return true
	}
	if valuesEqual(v_name, "head") {
		return true
	}
	if valuesEqual(v_name, "tail") {
		return true
	}
	if valuesEqual(v_name, "range") {
		return true
	}
	if valuesEqual(v_name, "concat") {
		return true
	}
	if valuesEqual(v_name, "lowercase") {
		return true
	}
	if valuesEqual(v_name, "split") {
		return true
	}
	if valuesEqual(v_name, "join") {
		return true
	}
	if valuesEqual(v_name, "trim") {
		return true
	}
	if valuesEqual(v_name, "contains") {
		return true
	}
	if valuesEqual(v_name, "replace") {
		return true
	}
	if valuesEqual(v_name, "starts_with") {
		return true
	}
	if valuesEqual(v_name, "ends_with") {
		return true
	}
	if valuesEqual(v_name, "substring") {
		return true
	}
	if valuesEqual(v_name, "char_at") {
		return true
	}
	if valuesEqual(v_name, "index") {
		return true
	}
	if valuesEqual(v_name, "mod") {
		return true
	}
	if valuesEqual(v_name, "to_string") {
		return true
	}
	if valuesEqual(v_name, "to_int") {
		return true
	}
	if valuesEqual(v_name, "sort") {
		return true
	}
	if valuesEqual(v_name, "slice") {
		return true
	}
	if valuesEqual(v_name, "reverse") {
		return true
	}
	if valuesEqual(v_name, "type_of") {
		return true
	}
	if valuesEqual(v_name, "get") {
		return true
	}
	if valuesEqual(v_name, "set") {
		return true
	}
	if valuesEqual(v_name, "keys") {
		return true
	}
	if valuesEqual(v_name, "values") {
		return true
	}
	if valuesEqual(v_name, "has_key") {
		return true
	}
	if valuesEqual(v_name, "merge") {
		return true
	}
	if valuesEqual(v_name, "remove") {
		return true
	}
	if valuesEqual(v_name, "read_file") {
		return true
	}
	if valuesEqual(v_name, "write_file") {
		return true
	}
	if valuesEqual(v_name, "file_exists") {
		return true
	}
	if valuesEqual(v_name, "append_file") {
		return true
	}
	if valuesEqual(v_name, "format") {
		return true
	}
	if valuesEqual(v_name, "ok") {
		return true
	}
	if valuesEqual(v_name, "err") {
		return true
	}
	if valuesEqual(v_name, "is_ok") {
		return true
	}
	if valuesEqual(v_name, "is_err") {
		return true
	}
	if valuesEqual(v_name, "unwrap") {
		return true
	}
	if valuesEqual(v_name, "unwrap_or") {
		return true
	}
	if valuesEqual(v_name, "unwrap_err") {
		return true
	}
	if valuesEqual(v_name, "try_read_file") {
		return true
	}
	if valuesEqual(v_name, "try_to_int") {
		return true
	}
	if valuesEqual(v_name, "map") {
		return true
	}
	if valuesEqual(v_name, "filter") {
		return true
	}
	if valuesEqual(v_name, "reduce") {
		return true
	}
	if valuesEqual(v_name, "push") {
		return true
	}
	if valuesEqual(v_name, "args") {
		return true
	}
	return false
}

func fn_push_scope(args ...any) any {
	v_scopes := args[0]
	return b_push(v_scopes, map[string]any{})
}

func fn_pop_scope(args ...any) any {
	v_scopes := args[0]
	return b_slice(v_scopes, int64(0), vSub(b_length(v_scopes), int64(1)))
}

func fn_define_local(args ...any) any {
	v_scopes := args[0]
	v_name := args[1]
	v_top := b_index(v_scopes, vSub(b_length(v_scopes), int64(1)))
	v_new_top := b_set(v_top, v_name, true)
	v_idx := vSub(b_length(v_scopes), int64(1))
	v_result := b_slice(v_scopes, int64(0), v_idx)
	return b_push(v_result, v_new_top)
}

func fn_is_local(args ...any) any {
	v_scopes := args[0]
	v_name := args[1]
	var v_i any = vSub(b_length(v_scopes), int64(1))
	for isTruthy(vGte(v_i, int64(0))) {
		if isTruthy(b_has_key(b_index(v_scopes, v_i), v_name)) {
			return true
		}
		v_i = vSub(v_i, int64(1))
	}
	return false
}

func fn_resolve_ident(args ...any) any {
	v_name := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	if isTruthy(fn_is_local(v_scopes, v_name)) {
		return vAdd("v_", v_name)
	}
	if isTruthy(b_has_key(v_user_fns, v_name)) {
		return vAdd("fn_", v_name)
	}
	if isTruthy(fn_is_builtin(v_name)) {
		return vAdd("b_", v_name)
	}
	return vAdd("v_", v_name)
}

func fn_make_indent(args ...any) any {
	v_level := args[0]
	var v_s any = ""
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, v_level)) {
		v_s = vAdd(v_s, "\t")
		v_i = vAdd(v_i, int64(1))
	}
	return v_s
}

func fn_go_quote(args ...any) any {
	v_s := args[0]
	var v_result any = "\""
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(v_s))) {
		v_ch := b_char_at(v_s, v_i)
		if isTruthy(vEq(v_ch, "\\")) {
			v_result = vAdd(v_result, "\\\\")
		} else {
			if isTruthy(vEq(v_ch, "\"")) {
				v_result = vAdd(v_result, "\\\"")
			} else {
				if isTruthy(vEq(v_ch, `
`)) {
					v_result = vAdd(v_result, "\\n")
				} else {
					if isTruthy(vEq(v_ch, "\t")) {
						v_result = vAdd(v_result, "\\t")
					} else {
						v_result = vAdd(v_result, v_ch)
					}
				}
			}
		}
		v_i = vAdd(v_i, int64(1))
	}
	return vAdd(v_result, "\"")
}

func fn_expr_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	v_t := b_get(v_node, "__type")
	if isTruthy(vEq(v_t, "IntLit")) {
		return vAdd(vAdd("int64(", b_to_string(b_get(v_node, "value"))), ")")
	}
	if isTruthy(vEq(v_t, "StringLit")) {
		if isTruthy(b_contains(b_get(v_node, "value"), `
`)) {
			return vAdd(vAdd("`", b_get(v_node, "value")), "`")
		}
		return fn_go_quote(b_get(v_node, "value"))
	}
	if isTruthy(vEq(v_t, "BoolLit")) {
		if isTruthy(b_get(v_node, "value")) {
			return "true"
		}
		return "false"
	}
	if isTruthy(vEq(v_t, "Ident")) {
		return fn_resolve_ident(b_get(v_node, "name"), v_scopes, v_user_fns)
	}
	if isTruthy(vEq(v_t, "BinaryExpr")) {
		v_left := fn_expr_string(b_get(v_node, "left"), v_scopes, v_user_fns, v_indent)
		v_right := fn_expr_string(b_get(v_node, "right"), v_scopes, v_user_fns, v_indent)
		return func() any {
			if valuesEqual(b_get(v_node, "op"), "+") {
				return vAdd(vAdd(vAdd(vAdd("vAdd(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "-") {
				return vAdd(vAdd(vAdd(vAdd("vSub(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "*") {
				return vAdd(vAdd(vAdd(vAdd("vMul(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "/") {
				return vAdd(vAdd(vAdd(vAdd("vDiv(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "==") {
				return vAdd(vAdd(vAdd(vAdd("vEq(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "!=") {
				return vAdd(vAdd(vAdd(vAdd("vNeq(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "<") {
				return vAdd(vAdd(vAdd(vAdd("vLt(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), ">") {
				return vAdd(vAdd(vAdd(vAdd("vGt(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), "<=") {
				return vAdd(vAdd(vAdd(vAdd("vLte(", v_left), ", "), v_right), ")")
			}
			if valuesEqual(b_get(v_node, "op"), ">=") {
				return vAdd(vAdd(vAdd(vAdd("vGte(", v_left), ", "), v_right), ")")
			}
			return "nil /* unknown op */"
		}()
	}
	if isTruthy(vEq(v_t, "CallExpr")) {
		return fn_call_expr_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	if isTruthy(vEq(v_t, "ArrayLit")) {
		if isTruthy(vEq(b_length(b_get(v_node, "elements")), int64(0))) {
			return "[]any{}"
		}
		var v_parts any = []any{}
		var v_i any = int64(0)
		for isTruthy(vLt(v_i, b_length(b_get(v_node, "elements")))) {
			v_parts = b_push(v_parts, fn_expr_string(b_index(b_get(v_node, "elements"), v_i), v_scopes, v_user_fns, v_indent))
			v_i = vAdd(v_i, int64(1))
		}
		return vAdd(vAdd("[]any{", b_join(v_parts, ", ")), "}")
	}
	if isTruthy(vEq(v_t, "HashLit")) {
		if isTruthy(vEq(b_length(b_get(v_node, "pairs")), int64(0))) {
			return "map[string]any{}"
		}
		var v_parts any = []any{}
		var v_i any = int64(0)
		for isTruthy(vLt(v_i, b_length(b_get(v_node, "pairs")))) {
			v_pair := b_index(b_get(v_node, "pairs"), v_i)
			v_parts = b_push(v_parts, vAdd(vAdd(fn_go_quote(b_get(v_pair, "key")), ": "), fn_expr_string(b_get(v_pair, "value"), v_scopes, v_user_fns, v_indent)))
			v_i = vAdd(v_i, int64(1))
		}
		return vAdd(vAdd("map[string]any{", b_join(v_parts, ", ")), "}")
	}
	if isTruthy(vEq(v_t, "DotExpr")) {
		return vAdd(vAdd(vAdd(vAdd("b_get(", fn_expr_string(b_get(v_node, "object"), v_scopes, v_user_fns, v_indent)), ", "), fn_go_quote(b_get(v_node, "field"))), ")")
	}
	if isTruthy(vEq(v_t, "FnLit")) {
		return fn_fn_lit_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	if isTruthy(vEq(v_t, "FanOutExpr")) {
		v_val := fn_expr_string(b_get(v_node, "value"), v_scopes, v_user_fns, v_indent)
		var v_fn_strs any = []any{}
		var v_i any = int64(0)
		for isTruthy(vLt(v_i, b_length(b_get(v_node, "fns")))) {
			v_fn_strs = b_push(v_fn_strs, fn_expr_string(b_index(b_get(v_node, "fns"), v_i), v_scopes, v_user_fns, v_indent))
			v_i = vAdd(v_i, int64(1))
		}
		return vAdd(vAdd(vAdd(vAdd("fanOut(", v_val), ", "), b_join(v_fn_strs, ", ")), ")")
	}
	if isTruthy(vEq(v_t, "IfExpr")) {
		return fn_if_expr_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	if isTruthy(vEq(v_t, "MatchExpr")) {
		return fn_match_expr_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	if isTruthy(vEq(v_t, "InterpLit")) {
		return fn_interp_lit_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	return vAdd(vAdd("nil /* unknown node: ", v_t), " */")
}

func fn_call_expr_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	var v_arg_strs any = []any{}
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_node, "args")))) {
		v_arg_strs = b_push(v_arg_strs, fn_expr_string(b_index(b_get(v_node, "args"), v_i), v_scopes, v_user_fns, v_indent))
		v_i = vAdd(v_i, int64(1))
	}
	v_arg_str := b_join(v_arg_strs, ", ")
	v_fn_type := b_get(b_get(v_node, "fn_node"), "__type")
	if isTruthy(vEq(v_fn_type, "Ident")) {
		v_name := b_get(b_get(v_node, "fn_node"), "name")
		if isTruthy(fn_is_local(v_scopes, v_name)) {
			v_resolved := fn_resolve_ident(v_name, v_scopes, v_user_fns)
			if isTruthy(vEq(b_length(v_arg_strs), int64(0))) {
				return vAdd(vAdd("callFn(", v_resolved), ")")
			}
			return vAdd(vAdd(vAdd(vAdd("callFn(", v_resolved), ", "), v_arg_str), ")")
		}
		v_resolved := fn_resolve_ident(v_name, v_scopes, v_user_fns)
		return vAdd(vAdd(vAdd(v_resolved, "("), v_arg_str), ")")
	}
	v_fn_expr := fn_expr_string(b_get(v_node, "fn_node"), v_scopes, v_user_fns, v_indent)
	if isTruthy(vEq(b_length(v_arg_strs), int64(0))) {
		return vAdd(vAdd("callFn(", v_fn_expr), ")")
	}
	return vAdd(vAdd(vAdd(vAdd("callFn(", v_fn_expr), ", "), v_arg_str), ")")
}

func fn_fn_lit_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	var v_result any = `func(args ...any) any {
`
	v_new_scopes := fn_push_scope(v_scopes)
	var v_s any = v_new_scopes
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_node, "params")))) {
		v_p := b_index(b_get(v_node, "params"), v_i)
		v_s = fn_define_local(v_s, b_get(v_p, "name"))
		v_result = vAdd(vAdd(vAdd(vAdd(vAdd(vAdd(v_result, fn_make_indent(vAdd(v_indent, int64(1)))), "v_"), b_get(v_p, "name")), " := args["), b_to_string(v_i)), `]
`)
		v_i = vAdd(v_i, int64(1))
	}
	v_result = vAdd(v_result, fn_emit_body_string(b_get(v_node, "body"), v_s, v_user_fns, vAdd(v_indent, int64(1))))
	v_result = vAdd(vAdd(v_result, fn_make_indent(v_indent)), "}")
	return v_result
}

func fn_if_expr_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	var v_result any = `func() any {
`
	v_result = vAdd(v_result, fn_emit_if_string(v_node, v_scopes, v_user_fns, vAdd(v_indent, int64(1)), true))
	v_result = vAdd(vAdd(v_result, fn_make_indent(v_indent)), "}()")
	return v_result
}

func fn_match_expr_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	var v_result any = `func() any {
`
	v_result = vAdd(v_result, fn_emit_match_return_string(v_node, v_scopes, v_user_fns, vAdd(v_indent, int64(1))))
	v_result = vAdd(vAdd(v_result, fn_make_indent(v_indent)), "}()")
	return v_result
}

func fn_interp_lit_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	var v_parts any = []any{}
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_node, "parts")))) {
		v_part := b_index(b_get(v_node, "parts"), v_i)
		v_pt := b_get(v_part, "__type")
		if isTruthy(vEq(v_pt, "StringLit")) {
			if isTruthy(vNeq(b_get(v_part, "value"), "")) {
				v_parts = b_push(v_parts, fn_go_quote(b_get(v_part, "value")))
			}
		} else {
			v_parts = b_push(v_parts, vAdd(vAdd("formatValue(", fn_expr_string(v_part, v_scopes, v_user_fns, v_indent)), ")"))
		}
		v_i = vAdd(v_i, int64(1))
	}
	if isTruthy(vEq(b_length(v_parts), int64(0))) {
		return "\"\""
	}
	if isTruthy(vEq(b_length(v_parts), int64(1))) {
		return b_index(v_parts, int64(0))
	}
	return b_join(v_parts, " + ")
}

func fn_emit_statement_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	v_t := b_get(v_node, "__type")
	v_ind := fn_make_indent(v_indent)
	if isTruthy(vEq(v_t, "LetStmt")) {
		v_val := fn_expr_string(b_get(v_node, "value"), v_scopes, v_user_fns, v_indent)
		if isTruthy(b_get(v_node, "mutable")) {
			return map[string]any{"code": vAdd(vAdd(vAdd(vAdd(vAdd(v_ind, "var v_"), b_get(v_node, "name")), " any = "), v_val), `
`), "scopes": fn_define_local(v_scopes, b_get(v_node, "name"))}
		}
		return map[string]any{"code": vAdd(vAdd(vAdd(vAdd(vAdd(v_ind, "v_"), b_get(v_node, "name")), " := "), v_val), `
`), "scopes": fn_define_local(v_scopes, b_get(v_node, "name"))}
	}
	if isTruthy(vEq(v_t, "AssignStmt")) {
		v_val := fn_expr_string(b_get(v_node, "value"), v_scopes, v_user_fns, v_indent)
		return map[string]any{"code": vAdd(vAdd(vAdd(vAdd(vAdd(v_ind, "v_"), b_get(v_node, "name")), " = "), v_val), `
`), "scopes": v_scopes}
	}
	if isTruthy(vEq(v_t, "WhileExpr")) {
		v_cond := fn_expr_string(b_get(v_node, "condition"), v_scopes, v_user_fns, v_indent)
		var v_code any = vAdd(vAdd(vAdd(v_ind, "for isTruthy("), v_cond), `) {
`)
		var v_i any = int64(0)
		var v_s any = v_scopes
		for isTruthy(vLt(v_i, b_length(b_get(v_node, "body")))) {
			v_r := fn_emit_statement_string(b_index(b_get(v_node, "body"), v_i), v_s, v_user_fns, vAdd(v_indent, int64(1)))
			v_code = vAdd(v_code, b_get(v_r, "code"))
			v_s = b_get(v_r, "scopes")
			v_i = vAdd(v_i, int64(1))
		}
		v_code = vAdd(vAdd(v_code, v_ind), `}
`)
		return map[string]any{"code": v_code, "scopes": v_scopes}
	}
	if isTruthy(vEq(v_t, "ReturnStmt")) {
		if isTruthy(vEq(b_get(v_node, "value"), int64(0))) {
			return map[string]any{"code": vAdd(v_ind, `return nil
`), "scopes": v_scopes}
		}
		v_val := fn_expr_string(b_get(v_node, "value"), v_scopes, v_user_fns, v_indent)
		return map[string]any{"code": vAdd(vAdd(vAdd(v_ind, "return "), v_val), `
`), "scopes": v_scopes}
	}
	if isTruthy(vEq(v_t, "IfExpr")) {
		return map[string]any{"code": fn_emit_if_string(v_node, v_scopes, v_user_fns, v_indent, false), "scopes": v_scopes}
	}
	v_val := fn_expr_string(v_node, v_scopes, v_user_fns, v_indent)
	return map[string]any{"code": vAdd(vAdd(vAdd(v_ind, "_ = "), v_val), `
`), "scopes": v_scopes}
}

func fn_emit_return_node_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	v_t := b_get(v_node, "__type")
	v_ind := fn_make_indent(v_indent)
	if isTruthy(vEq(v_t, "LetStmt")) {
		v_r := fn_emit_statement_string(v_node, v_scopes, v_user_fns, v_indent)
		return vAdd(vAdd(b_get(v_r, "code"), v_ind), `return nil
`)
	}
	if isTruthy(vEq(v_t, "AssignStmt")) {
		v_r := fn_emit_statement_string(v_node, v_scopes, v_user_fns, v_indent)
		return vAdd(vAdd(b_get(v_r, "code"), v_ind), `return nil
`)
	}
	if isTruthy(vEq(v_t, "WhileExpr")) {
		v_r := fn_emit_statement_string(v_node, v_scopes, v_user_fns, v_indent)
		return vAdd(vAdd(b_get(v_r, "code"), v_ind), `return nil
`)
	}
	if isTruthy(vEq(v_t, "ReturnStmt")) {
		v_r := fn_emit_statement_string(v_node, v_scopes, v_user_fns, v_indent)
		return b_get(v_r, "code")
	}
	if isTruthy(vEq(v_t, "IfExpr")) {
		return fn_emit_if_string(v_node, v_scopes, v_user_fns, v_indent, true)
	}
	if isTruthy(vEq(v_t, "MatchExpr")) {
		return fn_emit_match_return_string(v_node, v_scopes, v_user_fns, v_indent)
	}
	return vAdd(vAdd(vAdd(v_ind, "return "), fn_expr_string(v_node, v_scopes, v_user_fns, v_indent)), `
`)
}

func fn_emit_body_string(args ...any) any {
	v_body := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	if isTruthy(vEq(b_length(v_body), int64(0))) {
		return vAdd(fn_make_indent(v_indent), `return nil
`)
	}
	var v_result any = ""
	var v_s any = v_scopes
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(v_body))) {
		v_is_last := vEq(v_i, vSub(b_length(v_body), int64(1)))
		if isTruthy(v_is_last) {
			v_result = vAdd(v_result, fn_emit_return_node_string(b_index(v_body, v_i), v_s, v_user_fns, v_indent))
		} else {
			v_r := fn_emit_statement_string(b_index(v_body, v_i), v_s, v_user_fns, v_indent)
			v_result = vAdd(v_result, b_get(v_r, "code"))
			v_s = b_get(v_r, "scopes")
		}
		v_i = vAdd(v_i, int64(1))
	}
	return v_result
}

func fn_emit_if_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	v_is_return := args[4]
	v_ind := fn_make_indent(v_indent)
	v_cond := fn_expr_string(b_get(v_node, "condition"), v_scopes, v_user_fns, v_indent)
	var v_result any = vAdd(vAdd(vAdd(v_ind, "if isTruthy("), v_cond), `) {
`)
	if isTruthy(v_is_return) {
		v_result = vAdd(v_result, fn_emit_body_string(b_get(v_node, "then_body"), v_scopes, v_user_fns, vAdd(v_indent, int64(1))))
	} else {
		var v_i any = int64(0)
		var v_s any = v_scopes
		for isTruthy(vLt(v_i, b_length(b_get(v_node, "then_body")))) {
			v_r := fn_emit_statement_string(b_index(b_get(v_node, "then_body"), v_i), v_s, v_user_fns, vAdd(v_indent, int64(1)))
			v_result = vAdd(v_result, b_get(v_r, "code"))
			v_s = b_get(v_r, "scopes")
			v_i = vAdd(v_i, int64(1))
		}
	}
	if isTruthy(vGt(b_length(b_get(v_node, "else_body")), int64(0))) {
		v_result = vAdd(vAdd(v_result, v_ind), `} else {
`)
		if isTruthy(v_is_return) {
			v_result = vAdd(v_result, fn_emit_body_string(b_get(v_node, "else_body"), v_scopes, v_user_fns, vAdd(v_indent, int64(1))))
		} else {
			var v_i any = int64(0)
			var v_s any = v_scopes
			for isTruthy(vLt(v_i, b_length(b_get(v_node, "else_body")))) {
				v_r := fn_emit_statement_string(b_index(b_get(v_node, "else_body"), v_i), v_s, v_user_fns, vAdd(v_indent, int64(1)))
				v_result = vAdd(v_result, b_get(v_r, "code"))
				v_s = b_get(v_r, "scopes")
				v_i = vAdd(v_i, int64(1))
			}
		}
	} else {
		if isTruthy(v_is_return) {
			v_result = vAdd(vAdd(v_result, v_ind), `} else {
`)
			v_result = vAdd(vAdd(v_result, fn_make_indent(vAdd(v_indent, int64(1)))), `return nil
`)
		}
	}
	return vAdd(vAdd(v_result, v_ind), `}
`)
}

func fn_is_wildcard(args ...any) any {
	v_node := args[0]
	if isTruthy(vEq(b_get(v_node, "__type"), "Ident")) {
		return vEq(b_get(v_node, "name"), "_")
	}
	return false
}

func fn_emit_match_return_string(args ...any) any {
	v_node := args[0]
	v_scopes := args[1]
	v_user_fns := args[2]
	v_indent := args[3]
	v_ind := fn_make_indent(v_indent)
	v_subject := fn_expr_string(b_get(v_node, "subject"), v_scopes, v_user_fns, v_indent)
	var v_result any = ""
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_node, "arms")))) {
		v_arm := b_index(b_get(v_node, "arms"), v_i)
		if isTruthy(fn_is_wildcard(b_get(v_arm, "pattern"))) {
			v_result = vAdd(vAdd(vAdd(vAdd(v_result, v_ind), "return "), fn_expr_string(b_get(v_arm, "body"), v_scopes, v_user_fns, v_indent)), `
`)
			return v_result
		}
		v_pt := b_get(b_get(v_arm, "pattern"), "__type")
		if isTruthy(vEq(v_pt, "TypePattern")) {
			v_result = vAdd(vAdd(vAdd(vAdd(vAdd(vAdd(v_result, v_ind), "if _tv, _tvOk := "), v_subject), ".(map[string]any); _tvOk && _tv[\"__type\"] == "), fn_go_quote(b_get(b_get(v_arm, "pattern"), "type_name"))), ` {
`)
			v_s2 := fn_define_local(v_scopes, b_get(b_get(v_arm, "pattern"), "binding"))
			v_result = vAdd(vAdd(vAdd(vAdd(vAdd(vAdd(v_result, fn_make_indent(vAdd(v_indent, int64(1)))), "v_"), b_get(b_get(v_arm, "pattern"), "binding")), " := "), v_subject), `
`)
			v_result = vAdd(vAdd(vAdd(vAdd(v_result, fn_make_indent(vAdd(v_indent, int64(1)))), "return "), fn_expr_string(b_get(v_arm, "body"), v_s2, v_user_fns, vAdd(v_indent, int64(1)))), `
`)
			v_result = vAdd(vAdd(v_result, v_ind), `}
`)
		} else {
			v_result = vAdd(vAdd(vAdd(vAdd(vAdd(vAdd(v_result, v_ind), "if valuesEqual("), v_subject), ", "), fn_expr_string(b_get(v_arm, "pattern"), v_scopes, v_user_fns, v_indent)), `) {
`)
			v_result = vAdd(vAdd(vAdd(vAdd(v_result, fn_make_indent(vAdd(v_indent, int64(1)))), "return "), fn_expr_string(b_get(v_arm, "body"), v_scopes, v_user_fns, vAdd(v_indent, int64(1)))), `
`)
			v_result = vAdd(vAdd(v_result, v_ind), `}
`)
		}
		v_i = vAdd(v_i, int64(1))
	}
	return vAdd(vAdd(v_result, v_ind), `return nil
`)
}

func fn_generate(args ...any) any {
	v_prog := args[0]
	var v_user_fns any = map[string]any{}
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_prog, "decls")))) {
		v_decl := b_index(b_get(v_prog, "decls"), v_i)
		v_dt := b_get(v_decl, "__type")
		if isTruthy(vEq(v_dt, "FnDecl")) {
			v_user_fns = b_set(v_user_fns, b_get(v_decl, "name"), true)
		}
		if isTruthy(vEq(v_dt, "StructDecl")) {
			v_user_fns = b_set(v_user_fns, b_get(v_decl, "name"), true)
		}
		v_i = vAdd(v_i, int64(1))
	}
	var v_result any = fn_get_preamble()
	v_i = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_prog, "decls")))) {
		v_decl := b_index(b_get(v_prog, "decls"), v_i)
		v_dt := b_get(v_decl, "__type")
		if isTruthy(vEq(v_dt, "FnDecl")) {
			v_result = vAdd(v_result, fn_emit_fn_decl(v_decl, v_user_fns))
		}
		if isTruthy(vEq(v_dt, "StructDecl")) {
			v_result = vAdd(v_result, fn_emit_struct_decl(v_decl))
		}
		v_i = vAdd(v_i, int64(1))
	}
	v_result = vAdd(v_result, `
func main() {
	fn_main()
}
`)
	return v_result
}

func fn_emit_fn_decl(args ...any) any {
	v_decl := args[0]
	v_user_fns := args[1]
	var v_scopes any = fn_push_scope([]any{})
	var v_result any = vAdd(vAdd(`
func fn_`, b_get(v_decl, "name")), `(args ...any) any {
`)
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_decl, "params")))) {
		v_p := b_index(b_get(v_decl, "params"), v_i)
		v_scopes = fn_define_local(v_scopes, b_get(v_p, "name"))
		v_result = vAdd(vAdd(vAdd(vAdd(vAdd(v_result, "\tv_"), b_get(v_p, "name")), " := args["), b_to_string(v_i)), `]
`)
		v_i = vAdd(v_i, int64(1))
	}
	v_result = vAdd(v_result, fn_emit_body_string(b_get(v_decl, "body"), v_scopes, v_user_fns, int64(1)))
	return vAdd(v_result, `}
`)
}

func fn_emit_struct_decl(args ...any) any {
	v_decl := args[0]
	var v_result any = vAdd(vAdd(`
func fn_`, b_get(v_decl, "name")), `(args ...any) any {
`)
	v_result = vAdd(v_result, "\treturn map[string]any{")
	v_result = vAdd(vAdd(v_result, "\"__type\": "), fn_go_quote(b_get(v_decl, "name")))
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(b_get(v_decl, "fields")))) {
		v_field := b_index(b_get(v_decl, "fields"), v_i)
		v_result = vAdd(vAdd(vAdd(vAdd(vAdd(v_result, ", "), fn_go_quote(v_field)), ": args["), b_to_string(v_i)), "]")
		v_i = vAdd(v_i, int64(1))
	}
	return vAdd(v_result, `}
}
`)
}

func fn_get_preamble(args ...any) any {
	return b_read_file("runtime/preamble.tmpl")
}

func fn_resolve_imports(args ...any) any {
	v_decls := args[0]
	v_imported := args[1]
	var v_result any = []any{}
	var v_seen any = v_imported
	var v_i any = int64(0)
	for isTruthy(vLt(v_i, b_length(v_decls))) {
		v_decl := b_index(v_decls, v_i)
		v_dt := b_get(v_decl, "__type")
		if isTruthy(vEq(v_dt, "ImportDecl")) {
			v_path := vAdd(b_get(v_decl, "path"), ".vyr")
			if isTruthy(vEq(b_has_key(v_seen, v_path), false)) {
				v_seen = b_set(v_seen, v_path, true)
				v_src := b_read_file(v_path)
				v_toks := fn_tokenize(v_src)
				v_r := fn_parse(v_toks)
				if isTruthy(vNeq(b_get(v_r, "error"), "")) {
					_ = b_print(vAdd(vAdd(vAdd("Import error (", v_path), "): "), b_get(v_r, "error")))
					return v_result
				}
				v_imported_prog := b_get(v_r, "node")
				v_inner := fn_resolve_imports(b_get(v_imported_prog, "decls"), v_seen)
				var v_j any = int64(0)
				for isTruthy(vLt(v_j, b_length(v_inner))) {
					v_result = b_push(v_result, b_index(v_inner, v_j))
					v_j = vAdd(v_j, int64(1))
				}
			}
		} else {
			v_result = b_push(v_result, v_decl)
		}
		v_i = vAdd(v_i, int64(1))
	}
	return v_result
}

func fn_main(args ...any) any {
	v_argv := b_args()
	if isTruthy(vLt(b_length(v_argv), int64(1))) {
		_ = b_print("Usage: vyrc <input.vyr> [output.go]")
		return nil
	}
	v_input_file := b_index(v_argv, int64(0))
	v_output_file := func() any {
		if isTruthy(vGt(b_length(v_argv), int64(1))) {
			return b_index(v_argv, int64(1))
		} else {
			if isTruthy(b_ends_with(v_input_file, ".vyr")) {
				return b_replace(v_input_file, ".vyr", ".go")
			} else {
				return vAdd(v_input_file, ".go")
			}
		}
	}()
	v_source := b_read_file(v_input_file)
	v_tokens := fn_tokenize(v_source)
	v_result := fn_parse(v_tokens)
	if isTruthy(vNeq(b_get(v_result, "error"), "")) {
		_ = b_print(vAdd("Parse error: ", b_get(v_result, "error")))
		return nil
	}
	v_prog := b_get(v_result, "node")
	v_resolved_decls := fn_resolve_imports(b_get(v_prog, "decls"), map[string]any{})
	v_resolved_prog := fn_Program(v_resolved_decls)
	v_go_code := fn_generate(v_resolved_prog)
	_ = b_write_file(v_output_file, v_go_code)
	return b_print(vAdd(vAdd(vAdd("Compiled ", v_input_file), " -> "), v_output_file))
}

func main() {
	fn_main()
}
