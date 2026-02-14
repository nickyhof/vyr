package vm

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/nickyhof/vyr/internal/compiler"
)

// BuiltinFn is the signature for built-in functions.
type BuiltinFn func(args ...interface{}) (interface{}, error)

// formatValue formats a Vyr value for display.
func formatValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch val := v.(type) {
	case []interface{}:
		parts := make([]string, len(val))
		for i, elem := range val {
			parts[i] = formatValue(elem)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
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
	case *compiler.Result:
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

func typeOf(v interface{}) string {
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
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "map"
	case *compiler.Result:
		return "result"
	default:
		return "unknown"
	}
}

// Builtins returns the default builtin function table (indexes match compiler.builtinNames).
func Builtins(w *strings.Builder) []BuiltinFn {
	return []BuiltinFn{
		// 0: print
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("print: expected 1 argument, got %d", len(args))
			}
			s := formatValue(args[0])
			if w != nil {
				w.WriteString(s)
				w.WriteString("\n")
			} else {
				fmt.Println(s)
			}
			return nil, nil
		},
		// 1: uppercase
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("uppercase: expected 1 argument, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("uppercase: expected string, got %T", args[0])
			}
			return strings.ToUpper(s), nil
		},
		// 2: length
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("length: expected 1 argument, got %d", len(args))
			}
			switch v := args[0].(type) {
			case []interface{}:
				return int64(len(v)), nil
			case string:
				return int64(len([]rune(v))), nil
			default:
				return nil, fmt.Errorf("length: expected array or string, got %T", args[0])
			}
		},
		// 3: head
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("head: expected 1 argument, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("head: expected array, got %T", args[0])
			}
			if len(arr) == 0 {
				return nil, fmt.Errorf("head: empty array")
			}
			return arr[0], nil
		},
		// 4: tail
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("tail: expected 1 argument, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("tail: expected array, got %T", args[0])
			}
			if len(arr) == 0 {
				return []interface{}{}, nil
			}
			result := make([]interface{}, len(arr)-1)
			copy(result, arr[1:])
			return result, nil
		},
		// 5: range
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("range: expected 1 argument, got %d", len(args))
			}
			n, ok := args[0].(int64)
			if !ok {
				return nil, fmt.Errorf("range: expected int, got %T", args[0])
			}
			result := make([]interface{}, n)
			for i := int64(0); i < n; i++ {
				result[i] = i
			}
			return result, nil
		},
		// 6: concat
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("concat: expected 2 arguments, got %d", len(args))
			}
			a, aOk := args[0].([]interface{})
			b, bOk := args[1].([]interface{})
			if !aOk || !bOk {
				return nil, fmt.Errorf("concat: expected two arrays")
			}
			result := make([]interface{}, len(a)+len(b))
			copy(result, a)
			copy(result[len(a):], b)
			return result, nil
		},
		// 7: lowercase
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("lowercase: expected 1 argument, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("lowercase: expected string, got %T", args[0])
			}
			return strings.ToLower(s), nil
		},
		// 8: split
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("split: expected 2 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("split: first argument must be string")
			}
			sep, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("split: second argument must be string")
			}
			parts := strings.Split(s, sep)
			result := make([]interface{}, len(parts))
			for i, p := range parts {
				result[i] = p
			}
			return result, nil
		},
		// 9: join
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("join: expected 2 arguments, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("join: first argument must be array")
			}
			sep, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("join: second argument must be string")
			}
			parts := make([]string, len(arr))
			for i, v := range arr {
				parts[i] = formatValue(v)
			}
			return strings.Join(parts, sep), nil
		},
		// 10: trim
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("trim: expected 1 argument, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("trim: expected string, got %T", args[0])
			}
			return strings.TrimSpace(s), nil
		},
		// 11: contains (polymorphic: string or array)
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("contains: expected 2 arguments, got %d", len(args))
			}
			switch haystack := args[0].(type) {
			case string:
				needle, ok := args[1].(string)
				if !ok {
					return false, nil
				}
				return strings.Contains(haystack, needle), nil
			case []interface{}:
				for _, elem := range haystack {
					if valuesEqual(elem, args[1]) {
						return true, nil
					}
				}
				return false, nil
			default:
				return nil, fmt.Errorf("contains: first argument must be string or array")
			}
		},
		// 12: replace
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("replace: expected 3 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("replace: first argument must be string")
			}
			old, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("replace: second argument must be string")
			}
			new_, ok := args[2].(string)
			if !ok {
				return nil, fmt.Errorf("replace: third argument must be string")
			}
			return strings.ReplaceAll(s, old, new_), nil
		},
		// 13: starts_with
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("starts_with: expected 2 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("starts_with: first argument must be string")
			}
			prefix, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("starts_with: second argument must be string")
			}
			return strings.HasPrefix(s, prefix), nil
		},
		// 14: ends_with
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("ends_with: expected 2 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("ends_with: first argument must be string")
			}
			suffix, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("ends_with: second argument must be string")
			}
			return strings.HasSuffix(s, suffix), nil
		},
		// 15: substring
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("substring: expected 3 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("substring: first argument must be string")
			}
			start, ok := args[1].(int64)
			if !ok {
				return nil, fmt.Errorf("substring: second argument must be int")
			}
			end, ok := args[2].(int64)
			if !ok {
				return nil, fmt.Errorf("substring: third argument must be int")
			}
			runes := []rune(s)
			if start < 0 {
				start = 0
			}
			if end > int64(len(runes)) {
				end = int64(len(runes))
			}
			if start >= end {
				return "", nil
			}
			return string(runes[start:end]), nil
		},
		// 16: char_at
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("char_at: expected 2 arguments, got %d", len(args))
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("char_at: first argument must be string")
			}
			idx, ok := args[1].(int64)
			if !ok {
				return nil, fmt.Errorf("char_at: second argument must be int")
			}
			runes := []rune(s)
			if idx < 0 || idx >= int64(len(runes)) {
				return nil, fmt.Errorf("char_at: index %d out of bounds (length %d)", idx, len(runes))
			}
			return string(runes[idx]), nil
		},
		// 17: index (array indexing)
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("index: expected 2 arguments, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("index: first argument must be array")
			}
			idx, ok := args[1].(int64)
			if !ok {
				return nil, fmt.Errorf("index: second argument must be int")
			}
			if idx < 0 || idx >= int64(len(arr)) {
				return nil, fmt.Errorf("index: %d out of bounds (length %d)", idx, len(arr))
			}
			return arr[idx], nil
		},
		// 18: mod
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("mod: expected 2 arguments, got %d", len(args))
			}
			a, ok1 := args[0].(int64)
			b, ok2 := args[1].(int64)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("mod: expected two integers")
			}
			if b == 0 {
				return nil, fmt.Errorf("mod: division by zero")
			}
			return a % b, nil
		},
		// 19: to_string
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("to_string: expected 1 argument, got %d", len(args))
			}
			return formatValue(args[0]), nil
		},
		// 20: to_int
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("to_int: expected 1 argument, got %d", len(args))
			}
			switch v := args[0].(type) {
			case int64:
				return v, nil
			case string:
				n, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("to_int: cannot parse %q as integer", v)
				}
				return n, nil
			case bool:
				if v {
					return int64(1), nil
				}
				return int64(0), nil
			default:
				return nil, fmt.Errorf("to_int: cannot convert %T to int", args[0])
			}
		},
		// 21: sort
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("sort: expected 1 argument, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("sort: expected array, got %T", args[0])
			}
			sorted := make([]interface{}, len(arr))
			copy(sorted, arr)
			sort.Slice(sorted, func(i, j int) bool {
				ai, aiOk := sorted[i].(int64)
				aj, ajOk := sorted[j].(int64)
				if aiOk && ajOk {
					return ai < aj
				}
				return formatValue(sorted[i]) < formatValue(sorted[j])
			})
			return sorted, nil
		},
		// 22: slice
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("slice: expected 3 arguments, got %d", len(args))
			}
			arr, ok := args[0].([]interface{})
			if !ok {
				return nil, fmt.Errorf("slice: first argument must be array")
			}
			start, ok := args[1].(int64)
			if !ok {
				return nil, fmt.Errorf("slice: second argument must be int")
			}
			end, ok := args[2].(int64)
			if !ok {
				return nil, fmt.Errorf("slice: third argument must be int")
			}
			if start < 0 {
				start = 0
			}
			if end > int64(len(arr)) {
				end = int64(len(arr))
			}
			if start >= end {
				return []interface{}{}, nil
			}
			result := make([]interface{}, end-start)
			copy(result, arr[start:end])
			return result, nil
		},
		// 23: reverse (polymorphic: string or array)
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("reverse: expected 1 argument, got %d", len(args))
			}
			switch v := args[0].(type) {
			case string:
				runes := []rune(v)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return string(runes), nil
			case []interface{}:
				result := make([]interface{}, len(v))
				for i, j := 0, len(v)-1; j >= 0; i, j = i+1, j-1 {
					result[i] = v[j]
				}
				return result, nil
			default:
				return nil, fmt.Errorf("reverse: expected string or array, got %T", args[0])
			}
		},
		// 24: type_of
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("type_of: expected 1 argument, got %d", len(args))
			}
			return typeOf(args[0]), nil
		},
		// 25: get (map, key) or (map, key, default)
		func(args ...interface{}) (interface{}, error) {
			if len(args) < 2 || len(args) > 3 {
				return nil, fmt.Errorf("get: expected 2-3 arguments, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("get: expected map, got %T", args[0])
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("get: expected string key, got %T", args[1])
			}
			if val, exists := m[key]; exists {
				return val, nil
			}
			if len(args) == 3 {
				return args[2], nil
			}
			return nil, nil
		},
		// 26: set (returns new map)
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("set: expected 3 arguments, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("set: expected map, got %T", args[0])
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("set: expected string key, got %T", args[1])
			}
			newMap := make(map[string]interface{}, len(m)+1)
			for k, v := range m {
				newMap[k] = v
			}
			newMap[key] = args[2]
			return newMap, nil
		},
		// 27: keys
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("keys: expected 1 argument, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("keys: expected map, got %T", args[0])
			}
			ks := make([]string, 0, len(m))
			for k := range m {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			result := make([]interface{}, len(ks))
			for i, k := range ks {
				result[i] = k
			}
			return result, nil
		},
		// 28: values
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("values: expected 1 argument, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("values: expected map, got %T", args[0])
			}
			ks := make([]string, 0, len(m))
			for k := range m {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			result := make([]interface{}, len(ks))
			for i, k := range ks {
				result[i] = m[k]
			}
			return result, nil
		},
		// 29: has_key
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("has_key: expected 2 arguments, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("has_key: expected map, got %T", args[0])
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("has_key: expected string key, got %T", args[1])
			}
			_, exists := m[key]
			return exists, nil
		},
		// 30: merge
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("merge: expected 2 arguments, got %d", len(args))
			}
			m1, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("merge: expected map, got %T", args[0])
			}
			m2, ok := args[1].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("merge: expected map, got %T", args[1])
			}
			result := make(map[string]interface{}, len(m1)+len(m2))
			for k, v := range m1 {
				result[k] = v
			}
			for k, v := range m2 {
				result[k] = v
			}
			return result, nil
		},
		// 31: remove
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("remove: expected 2 arguments, got %d", len(args))
			}
			m, ok := args[0].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("remove: expected map, got %T", args[0])
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("remove: expected string key, got %T", args[1])
			}
			result := make(map[string]interface{}, len(m))
			for k, v := range m {
				if k != key {
					result[k] = v
				}
			}
			return result, nil
		},
		// 32: read_file
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("read_file: expected 1 argument, got %d", len(args))
			}
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("read_file: expected string path, got %T", args[0])
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read_file: %v", err)
			}
			return string(data), nil
		},
		// 33: write_file
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("write_file: expected 2 arguments, got %d", len(args))
			}
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("write_file: expected string path, got %T", args[0])
			}
			content, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("write_file: expected string content, got %T", args[1])
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("write_file: %v", err)
			}
			return content, nil
		},
		// 34: file_exists
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("file_exists: expected 1 argument, got %d", len(args))
			}
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("file_exists: expected string path, got %T", args[0])
			}
			_, err := os.Stat(path)
			return err == nil, nil
		},
		// 35: append_file
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("append_file: expected 2 arguments, got %d", len(args))
			}
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("append_file: expected string path, got %T", args[0])
			}
			content, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("append_file: expected string content, got %T", args[1])
			}
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("append_file: %v", err)
			}
			defer func() { _ = f.Close() }()
			if _, err := f.WriteString(content); err != nil {
				return nil, fmt.Errorf("append_file: %v", err)
			}
			return content, nil
		},
		// 36: format
		func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("format: expected at least 1 argument")
			}
			tmpl, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("format: expected string template, got %T", args[0])
			}
			result := tmpl
			for i := 1; i < len(args); i++ {
				idx := strings.Index(result, "{}")
				if idx == -1 {
					break
				}
				result = result[:idx] + formatValue(args[i]) + result[idx+2:]
			}
			return result, nil
		},
		// 37: ok
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("ok: expected 1 argument, got %d", len(args))
			}
			return &compiler.Result{Ok: true, Value: args[0]}, nil
		},
		// 38: err
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("err: expected 1 argument, got %d", len(args))
			}
			return &compiler.Result{Ok: false, Value: args[0]}, nil
		},
		// 39: is_ok
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("is_ok: expected 1 argument, got %d", len(args))
			}
			r, ok := args[0].(*compiler.Result)
			if !ok {
				return false, nil
			}
			return r.Ok, nil
		},
		// 40: is_err
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("is_err: expected 1 argument, got %d", len(args))
			}
			r, ok := args[0].(*compiler.Result)
			if !ok {
				return false, nil
			}
			return !r.Ok, nil
		},
		// 41: unwrap
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("unwrap: expected 1 argument, got %d", len(args))
			}
			r, ok := args[0].(*compiler.Result)
			if !ok {
				return nil, fmt.Errorf("unwrap: expected result, got %T", args[0])
			}
			if !r.Ok {
				return nil, fmt.Errorf("unwrap called on Err: %v", r.Value)
			}
			return r.Value, nil
		},
		// 42: unwrap_or
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("unwrap_or: expected 2 arguments, got %d", len(args))
			}
			r, ok := args[0].(*compiler.Result)
			if !ok {
				return args[1], nil
			}
			if r.Ok {
				return r.Value, nil
			}
			return args[1], nil
		},
		// 43: try_read_file
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("try_read_file: expected 1 argument, got %d", len(args))
			}
			path, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("try_read_file: expected string path, got %T", args[0])
			}
			data, fileErr := os.ReadFile(path)
			if fileErr != nil {
				return &compiler.Result{Ok: false, Value: fileErr.Error()}, nil
			}
			return &compiler.Result{Ok: true, Value: string(data)}, nil
		},
		// 44: try_to_int
		func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("try_to_int: expected 1 argument, got %d", len(args))
			}
			switch v := args[0].(type) {
			case int64:
				return &compiler.Result{Ok: true, Value: v}, nil
			case string:
				n, parseErr := strconv.ParseInt(v, 10, 64)
				if parseErr != nil {
					return &compiler.Result{Ok: false, Value: fmt.Sprintf("cannot parse '%s' as int", v)}, nil
				}
				return &compiler.Result{Ok: true, Value: n}, nil
			case bool:
				if v {
					return &compiler.Result{Ok: true, Value: int64(1)}, nil
				}
				return &compiler.Result{Ok: true, Value: int64(0)}, nil
			default:
				return &compiler.Result{Ok: false, Value: fmt.Sprintf("cannot convert %T to int", args[0])}, nil
			}
		},
	}
}
