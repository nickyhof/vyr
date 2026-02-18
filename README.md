# Vyr

A functional, pipe-first programming language that transpiles to Go.

Vyr emphasizes data flow through pipes (`|>`), closures, pattern matching, and a growing standard library — all compiled to Go for native performance. The compiler is **self-hosted**: written in Vyr itself.

## Quick Start

```bash
# Build the compiler from source
go build -o vyrc bootstrap/vyrc.go

# Run a program
./vyr run examples/hello.vyr

# Compile to a native binary
./vyr build examples/hello.vyr -o hello
./hello

# Or use vyrc directly
./vyrc examples/hello.vyr hello.go
go run hello.go
```

## Features

- **Pipe operator** — chain transformations naturally: `5 |> double |> print`
- **String interpolation** — `"hello ${name}, you are ${age} years old"`
- **First-class functions & closures** — `let add5 = make_adder(5)`
- **Pattern matching** — `x |> match { 0 => "zero" _ => "other" }`
- **Structs** — `struct Point { x, y }` with dot access
- **Fan-out** — broadcast a value: `x |> [f, g, h]`
- **Hashmaps** — `#{name: "alice", age: 30}` with dot access
- **Result type** — `ok(val)`, `err(msg)` for safe error handling
- **Mutable variables** — `let mut x = 0`
- **While loops** — `while x < 10 { ... }`
- **Return statements** — `return value`
- **Standard library** — 128 functions across 12 modules
- **Self-hosted compiler** — the compiler is written in Vyr

## Standard Library

> 📖 **[Full API Reference](https://nickyhof.github.io/vyr/)** — searchable docs with every function signature and description.

| Module | Functions | Description |
|--------|-----------|-------------|
| `std/math` | 19 | `abs`, `max`, `clamp`, `fib`, `gcd`, `pow`, … |
| `std/string` | 21 | `repeat`, `pad_left`, `trim`, `contains`, `replace`, … |
| `std/collections` | 27 | `any`, `all`, `zip`, `group_by`, `sort_by`, `chunk`, … |
| `std/result` | 9 | `map_ok`, `and_then`, `flatten_result`, `collect_results`, … |
| `std/functional` | 7 | `compose`, `pipe`, `identity`, `always`, `flip`, … |
| `std/io` | 3 | `read_lines`, `write_lines`, `append_line` |
| `std/http` | 9 | `get_json`, `post_json`, `json_response`, `serve`, … |
| `std/fs` | 9 | `ensure_dir`, `list_files`, `read_json`, `write_json`, `copy_file`, … |
| `std/env` | 5 | `env_or`, `require_env`, `home_dir`, `path_dirs`, … |
| `std/time` | 7 | `timestamp`, `today`, `elapsed_ms`, `sleep`, `format_time`, … |
| `std/regex` | 7 | `is_match`, `extract`, `replace_all`, `is_email`, `is_numeric`, … |
| `std/process` | 5 | `run`, `run_output`, `run_or_exit`, `shell`, `cwd` |

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
// Structs + pattern matching
struct Circle { center, radius }
struct Rect { origin, width, height }

fn area(shape) {
  shape |> match {
    Circle(c) => c.radius * c.radius * 3
    Rect(r) => r.width * r.height
    _ => 0
  }
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
bootstrap/         Self-hosted compiler (written in Vyr)
  lexer.vyr          Tokenizer
  parser.vyr         Recursive-descent parser
  codegen.vyr        Go code generator
  main.vyr           Compiler entry point
  vyrc.go            Compiled Go source (checked in for bootstrapping)
runtime/           Go runtime template embedded in generated programs
std/               Standard library (.vyr files)
examples/          Example programs
vyr                Shell script CLI (run, build, compile)
```

## Development

```bash
make bootstrap   # Rebuild vyrc from Vyr source (requires existing vyrc)
make test        # Compile and run example programs
make clean       # Remove built binaries
```

### Bootstrapping

The compiler compiles itself. `bootstrap/vyrc.go` is the compiled Go form of the compiler, checked into the repo so you can always build from a fresh clone:

```bash
go build -o vyrc bootstrap/vyrc.go   # Build compiler from checked-in source
make bootstrap                       # Rebuild using the Vyr source files
```

## License

Apache 2.0