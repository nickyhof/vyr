package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nickyhof/vyr/internal/lexer"
	"github.com/nickyhof/vyr/internal/parser"
)

func runProgram(t *testing.T, input string) string {
	t.Helper()
	tokens := lexer.New(input).Tokenize()
	prog, err := parser.New(tokens).Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	goCode := Generate(prog)

	tmpDir, err := os.MkdirTemp("", "vyr-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("failed to write generated code: %v", err)
	}

	cmd := exec.Command("go", "run", mainFile)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run failed: %v\nstderr: %s\ngenerated code:\n%s", err, stderr.String(), goCode)
	}

	return strings.TrimSpace(stdout.String())
}

func expect(t *testing.T, input, want string) {
	t.Helper()
	got := runProgram(t, input)
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// --- Milestone 1: Pipes ---
func TestEndToEndHelloWorld(t *testing.T) {
	expect(t, `fn main() { "hello world" |> uppercase |> print }`, "HELLO WORLD")
}

// --- Milestone 2: Variables & Arithmetic ---
func TestArithmetic(t *testing.T) {
	expect(t, `fn main() { 2 + 3 * 4 |> print }`, "14")
}
func TestLetBinding(t *testing.T) {
	expect(t, `fn main() { let x = 10 + 5
  x |> print }`, "15")
}
func TestUserFunction(t *testing.T) {
	expect(t, `fn double(x) { x * 2 }
fn main() { 21 |> double |> print }`, "42")
}
func TestUserFunctionWithLet(t *testing.T) {
	expect(t, `fn double(x) { x * 2 }
fn main() { let result = 21 |> double
  result |> print }`, "42")
}
func TestMultipleFunctions(t *testing.T) {
	expect(t, `fn add_one(x) { x + 1 }
fn double(x) { x * 2 }
fn main() { 5 |> double |> add_one |> print }`, "11")
}

// --- Milestone 3: Data-Flow ---
func TestArrayLiteral(t *testing.T) {
	expect(t, `fn main() { [1, 2, 3] |> print }`, "[1, 2, 3]")
}
func TestFanOut(t *testing.T) {
	expect(t, `fn double(x) { x * 2 }
fn add_ten(x) { x + 10 }
fn main() { 5 |> [double, add_ten] |> print }`, "[10, 15]")
}
func TestLambda(t *testing.T) {
	expect(t, `fn main() { let sq = fn(x) { x * x }
  9 |> sq |> print }`, "81")
}
func TestMapWithLambda(t *testing.T) {
	expect(t, `fn main() { [1, 2, 3, 4, 5] |> map(fn(x) { x * x }) |> print }`, "[1, 4, 9, 16, 25]")
}
func TestMatchLiteral(t *testing.T) {
	expect(t, `fn main() { 42 |> match { 0 => "zero" 42 => "the answer" _ => "?" } |> print }`, "the answer")
}
func TestMatchWildcard(t *testing.T) {
	expect(t, `fn main() { 99 |> match { 42 => "the answer" _ => "other" } |> print }`, "other")
}
func TestMatchWithString(t *testing.T) {
	expect(t, `fn main() { "hello" |> match { "goodbye" => "bye" "hello" => "hi" _ => "?" } |> print }`, "hi")
}
func TestMapWithNamedFunction(t *testing.T) {
	expect(t, `fn double(x) { x * 2 }
fn main() { [1, 2, 3] |> map(double) |> print }`, "[2, 4, 6]")
}

// --- Milestone 4: Collections & Types ---
func TestBooleans(t *testing.T) {
	expect(t, `fn main() { true |> print }`, "true")
	expect(t, `fn main() { false |> print }`, "false")
}
func TestComparisonEq(t *testing.T) {
	expect(t, `fn main() { 5 == 5 |> print }`, "true")
	expect(t, `fn main() { 5 == 3 |> print }`, "false")
}
func TestComparisonNeq(t *testing.T) {
	expect(t, `fn main() { 5 != 3 |> print }`, "true")
}
func TestComparisonLtGt(t *testing.T) {
	expect(t, `fn main() { 3 < 5 |> print }`, "true")
	expect(t, `fn main() { 5 > 3 |> print }`, "true")
	expect(t, `fn main() { 5 <= 5 |> print }`, "true")
	expect(t, `fn main() { 5 >= 6 |> print }`, "false")
}
func TestIfElse(t *testing.T) {
	expect(t, `fn main() { if 5 > 3 { "yes" } else { "no" } |> print }`, "yes")
	expect(t, `fn main() { if 3 > 5 { "yes" } else { "no" } |> print }`, "no")
}
func TestIfWithoutElse(t *testing.T) {
	expect(t, `fn main() { let x = if true { 42 }
  x |> print }`, "42")
}
func TestStringConcat(t *testing.T) {
	expect(t, `fn main() { "Hello, " + "World!" |> print }`, "Hello, World!")
}
func TestFilter(t *testing.T) {
	expect(t, `fn main() {
  [1, 2, 3, 4, 5, 6] |> filter(fn(x) { x > 3 }) |> print
}`, "[4, 5, 6]")
}
func TestReduce(t *testing.T) {
	expect(t, `fn main() {
  [1, 2, 3, 4, 5] |> reduce(0, fn(acc, x) { acc + x }) |> print
}`, "15")
}
func TestLength(t *testing.T) {
	expect(t, `fn main() { [1, 2, 3] |> length |> print }`, "3")
}
func TestHeadTail(t *testing.T) {
	expect(t, `fn main() { [10, 20, 30] |> head |> print }`, "10")
	expect(t, `fn main() { [10, 20, 30] |> tail |> print }`, "[20, 30]")
}
func TestRange(t *testing.T) {
	expect(t, `fn main() { 5 |> range |> print }`, "[0, 1, 2, 3, 4]")
}
func TestConcat(t *testing.T) {
	expect(t, `fn main() { concat([1, 2], [3, 4]) |> print }`, "[1, 2, 3, 4]")
}
func TestTypedFunction(t *testing.T) {
	expect(t, `fn double(x: int): int { x * 2 }
fn main() { 21 |> double |> print }`, "42")
}
func TestAbsoluteValue(t *testing.T) {
	expect(t, `fn abs(x: int): int { if x >= 0 { x } else { 0 - x } }
fn main() { 0 - 42 |> abs |> print }`, "42")
}
func TestFilterMapChain(t *testing.T) {
	expect(t, `fn main() {
  10 |> range
    |> filter(fn(x) { x > 0 })
    |> filter(fn(x) { x * x < 50 })
    |> map(fn(x) { x * x })
    |> print
}`, "[1, 4, 9, 16, 25, 36, 49]")
}

// --- Milestone 6A: New builtins ---
func TestLowercase(t *testing.T) {
	expect(t, `fn main() { "HELLO" |> lowercase |> print }`, "hello")
}
func TestSplit(t *testing.T) {
	expect(t, `fn main() { "a,b,c" |> split(",") |> print }`, "[a, b, c]")
}
func TestJoin(t *testing.T) {
	expect(t, `fn main() { ["x", "y", "z"] |> join("-") |> print }`, "x-y-z")
}
func TestTrim(t *testing.T) {
	expect(t, `fn main() { "  hello  " |> trim |> print }`, "hello")
}
func TestContainsString(t *testing.T) {
	expect(t, `fn main() { contains("hello world", "world") |> print }`, "true")
	expect(t, `fn main() { contains("hello", "xyz") |> print }`, "false")
}
func TestContainsArray(t *testing.T) {
	expect(t, `fn main() { contains([1, 2, 3], 2) |> print }`, "true")
	expect(t, `fn main() { contains([1, 2, 3], 5) |> print }`, "false")
}
func TestReplace(t *testing.T) {
	expect(t, `fn main() { "hello world" |> replace("world", "vyr") |> print }`, "hello vyr")
}
func TestStartsEndsWith(t *testing.T) {
	expect(t, `fn main() { starts_with("hello", "hel") |> print }`, "true")
	expect(t, `fn main() { ends_with("hello", "llo") |> print }`, "true")
}
func TestSubstring(t *testing.T) {
	expect(t, `fn main() { substring("hello world", 6, 11) |> print }`, "world")
}
func TestCharAt(t *testing.T) {
	expect(t, `fn main() { char_at("abc", 1) |> print }`, "b")
}
func TestIndex(t *testing.T) {
	expect(t, `fn main() { index([10, 20, 30], 2) |> print }`, "30")
}
func TestMod(t *testing.T) {
	expect(t, `fn main() { mod(10, 3) |> print }`, "1")
	expect(t, `fn main() { mod(8, 2) |> print }`, "0")
}
func TestToString(t *testing.T) {
	expect(t, `fn main() { 42 |> to_string |> print }`, "42")
	expect(t, `fn main() { true |> to_string |> print }`, "true")
}
func TestToInt(t *testing.T) {
	expect(t, `fn main() { "123" |> to_int |> print }`, "123")
}
func TestSort(t *testing.T) {
	expect(t, `fn main() { [3, 1, 4, 1, 5, 9] |> sort |> print }`, "[1, 1, 3, 4, 5, 9]")
}
func TestSlice(t *testing.T) {
	expect(t, `fn main() { slice([10, 20, 30, 40, 50], 1, 4) |> print }`, "[20, 30, 40]")
}
func TestReverseString(t *testing.T) {
	expect(t, `fn main() { "hello" |> reverse |> print }`, "olleh")
}
func TestReverseArray(t *testing.T) {
	expect(t, `fn main() { [1, 2, 3] |> reverse |> print }`, "[3, 2, 1]")
}
func TestTypeOf(t *testing.T) {
	expect(t, `fn main() { 42 |> type_of |> print }`, "int")
	expect(t, `fn main() { "hi" |> type_of |> print }`, "string")
	expect(t, `fn main() { true |> type_of |> print }`, "bool")
	expect(t, `fn main() { [1] |> type_of |> print }`, "array")
}
func TestEscapeChars(t *testing.T) {
	expect(t, `fn main() { "line1\nline2" |> print }`, "line1\nline2")
	expect(t, `fn main() { "tab\there" |> print }`, "tab\there")
}
func TestRecursion(t *testing.T) {
	expect(t, `fn factorial(n) {
  if n <= 1 { 1 } else { n * factorial(n - 1) }
}
fn main() { 5 |> factorial |> print }`, "120")
}

// --- Milestone 6B: Closures ---
func TestClosureCapture(t *testing.T) {
	expect(t, `fn make_adder(n) {
  fn(x) { x + n }
}
fn main() {
  let add5 = make_adder(5)
  10 |> add5 |> print
}`, "15")
}

func TestClosureInMap(t *testing.T) {
	expect(t, `fn main() {
  let factor = 3
  [1, 2, 3] |> map(fn(x) { x * factor }) |> print
}`, "[3, 6, 9]")
}

func TestClosureMultiCapture(t *testing.T) {
	expect(t, `fn main() {
  let a = 10
  let b = 20
  [1, 2, 3] |> map(fn(x) { x + a + b }) |> print
}`, "[31, 32, 33]")
}

func TestClosureInFilter(t *testing.T) {
	expect(t, `fn main() {
  let threshold = 3
  [1, 2, 3, 4, 5] |> filter(fn(x) { x > threshold }) |> print
}`, "[4, 5]")
}

func TestClosureWithBuiltins(t *testing.T) {
	expect(t, `fn main() {
  let s = "abc"
  let len = s |> length
  len |> range |> map(fn(i) { char_at(s, i) }) |> print
}`, "[a, b, c]")
}

func TestClosureWithIndex(t *testing.T) {
	expect(t, `fn main() {
  let arr = [10, 20, 30]
  let len = arr |> length
  len |> range |> map(fn(i) { index(arr, i) }) |> print
}`, "[10, 20, 30]")
}

// --- Milestone 7: Hashmaps ---
func TestHashLiteral(t *testing.T) {
	expect(t, `fn main() { #{name: "alice", age: 30} |> print }`, `#{age: 30, name: alice}`)
}

func TestHashGet(t *testing.T) {
	expect(t, `fn main() { get(#{name: "alice"}, "name") |> print }`, "alice")
}

func TestHashDotAccess(t *testing.T) {
	expect(t, `fn main() {
  let user = #{name: "bob", age: 25}
  user.name |> print
}`, "bob")
}

func TestHashSet(t *testing.T) {
	expect(t, `fn main() {
  let m = #{x: 1}
  let m2 = set(m, "y", 2)
  m2 |> print
}`, `#{x: 1, y: 2}`)
}

func TestHashKeys(t *testing.T) {
	expect(t, `fn main() { #{b: 2, a: 1} |> keys |> print }`, "[a, b]")
}

func TestHashValues(t *testing.T) {
	expect(t, `fn main() { #{b: 2, a: 1} |> values |> print }`, "[1, 2]")
}

func TestHashHasKey(t *testing.T) {
	expect(t, `fn main() { has_key(#{x: 1}, "x") |> print }`, "true")
	expect(t, `fn main() { has_key(#{x: 1}, "y") |> print }`, "false")
}

func TestHashMerge(t *testing.T) {
	expect(t, `fn main() { merge(#{a: 1}, #{b: 2}) |> print }`, `#{a: 1, b: 2}`)
}

func TestHashRemove(t *testing.T) {
	expect(t, `fn main() { remove(#{a: 1, b: 2}, "a") |> print }`, `#{b: 2}`)
}

func TestHashTypeOf(t *testing.T) {
	expect(t, `fn main() { #{} |> type_of |> print }`, "map")
}

func TestHashPipe(t *testing.T) {
	expect(t, `fn main() {
  #{name: "alice", score: 95}
    |> get("name")
    |> uppercase
    |> print
}`, "ALICE")
}

// --- Milestone 9: String Format ---
func TestFormat(t *testing.T) {
	expect(t, `fn main() { format("hello {}, you are {}", "alice", 30) |> print }`, "hello alice, you are 30")
}

func TestFormatNoArgs(t *testing.T) {
	expect(t, `fn main() { format("plain text") |> print }`, "plain text")
}

// --- Milestone 11: Error Handling ---
func TestResultOk(t *testing.T) {
	expect(t, `fn main() { ok(42) |> print }`, "Ok(42)")
}

func TestResultErr(t *testing.T) {
	expect(t, `fn main() { err("oops") |> print }`, "Err(oops)")
}

func TestIsOk(t *testing.T) {
	expect(t, `fn main() { ok(1) |> is_ok |> print }`, "true")
	expect(t, `fn main() { err("x") |> is_ok |> print }`, "false")
}

func TestIsErr(t *testing.T) {
	expect(t, `fn main() { err("x") |> is_err |> print }`, "true")
	expect(t, `fn main() { ok(1) |> is_err |> print }`, "false")
}

func TestUnwrap(t *testing.T) {
	expect(t, `fn main() { ok(42) |> unwrap |> print }`, "42")
}

func TestUnwrapOr(t *testing.T) {
	expect(t, `fn main() { ok(42) |> unwrap_or(0) |> print }`, "42")
	expect(t, `fn main() { err("x") |> unwrap_or(0) |> print }`, "0")
}

func TestTryToInt(t *testing.T) {
	expect(t, `fn main() { "42" |> try_to_int |> print }`, "Ok(42)")
	expect(t, `fn main() { "abc" |> try_to_int |> print }`, "Err(cannot parse 'abc' as int)")
}

func TestResultTypeOf(t *testing.T) {
	expect(t, `fn main() { ok(1) |> type_of |> print }`, "result")
}

func TestResultSafeDivide(t *testing.T) {
	expect(t, `fn safe_divide(a, b) {
  if b == 0 { err("division by zero") } else { ok(a / b) }
}
fn main() {
  safe_divide(10, 2) |> unwrap |> print
  safe_divide(10, 0) |> unwrap_or(0) |> print
}`, "5\n0")
}

// --- String Interpolation ---
func TestInterpBasic(t *testing.T) {
	expect(t, `fn main() {
  let name = "world"
  "hello ${name}" |> print
}`, "hello world")
}

func TestInterpExpression(t *testing.T) {
	expect(t, `fn main() { "two plus three is ${2 + 3}" |> print }`, "two plus three is 5")
}

func TestInterpMultiple(t *testing.T) {
	expect(t, `fn main() {
  let a = "foo"
  let b = "bar"
  "${a} and ${b}" |> print
}`, "foo and bar")
}

func TestInterpNestedCall(t *testing.T) {
	expect(t, `fn main() {
  let name = "alice"
  "hello ${uppercase(name)}" |> print
}`, "hello ALICE")
}

func TestInterpEscaped(t *testing.T) {
	expect(t, `fn main() { "price: \${42}" |> print }`, "price: ${42}")
}

func TestInterpPlainString(t *testing.T) {
	expect(t, `fn main() { "no interpolation here" |> print }`, "no interpolation here")
}

func TestInterpOnlyExpr(t *testing.T) {
	expect(t, `fn main() { "${42}" |> print }`, "42")
}

func TestInterpWithPipe(t *testing.T) {
	expect(t, `fn main() {
  let x = 10
  "value is ${x}" |> uppercase |> print
}`, "VALUE IS 10")
}

func TestInterpIntCoercion(t *testing.T) {
	expect(t, `fn main() {
  let n = 42
  "the answer is ${n}" |> print
}`, "the answer is 42")
}
