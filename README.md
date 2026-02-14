# Vyr

A functional, pipe-first programming language built in Go.

Vyr emphasizes data flow through pipes (`|>`), closures, pattern matching, and a growing standard library — all compiled to bytecode and run on a custom VM.

## Quick Start

```bash
# Build
make build

# Run a program
./vyr run examples/hello.vyr

# Interactive REPL
./vyr repl
```

## Features

- **Pipe operator** — chain transformations naturally: `5 |> double |> print`
- **String interpolation** — `"hello ${name}, you are ${age} years old"`
- **First-class functions & closures** — `let add5 = make_adder(5)`
- **Pattern matching** — `x |> match { 0 => "zero" _ => "other" }`
- **Fan-out** — broadcast a value: `x |> [f, g, h]`
- **Hashmaps** — `#{name: "alice", age: 30}` with dot access
- **Result type** — `ok(val)`, `err(msg)` for safe error handling
- **Gradual typing** — optional type annotations: `fn double(x: int): int { x * 2 }`
- **Standard library** — `std/math`, `std/string`, `std/collections`, `std/io`, `std/result`

## Examples

```
// Factorial with pipes
fn factorial(n) {
  if n <= 1 { 1 } else { n * factorial(n - 1) }
}

fn main() {
  5 |> factorial |> print  // 120
}
```

```
// String interpolation + closures
fn greet(greeting) {
  fn(name) { "${greeting}, ${name}!" }
}

fn main() {
  let hello = greet("Hello")
  hello("world") |> print  // Hello, world!
}
```

```
// Data pipeline
fn main() {
  [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    |> filter(fn(x) { mod(x, 2) == 0 })
    |> map(fn(x) { x * x })
    |> print  // [4, 16, 36, 64, 100]
}
```

## Project Structure

```
cmd/vyr/       CLI entrypoint (run, repl)
internal/      Compiler pipeline
  lexer/         Tokenizer
  parser/        AST construction
  checker/       Gradual type checker
  compiler/      Bytecode compiler
  vm/            Virtual machine
  loader/        Import resolution
std/           Standard library (.vyr files)
examples/      Example programs
```

## Development

```bash
make test       # Run all tests
make cover      # Generate coverage report
make fmt        # Format code
make vet        # Vet code
make lint       # Run staticcheck
make check      # fmt + vet + test
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
