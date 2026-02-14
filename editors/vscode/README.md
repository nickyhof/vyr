# Vyr for VS Code

Syntax highlighting for the [Vyr](https://github.com/nickyhof/vyr) programming language.

## Features

- Syntax highlighting for all Vyr constructs
- String interpolation (`${expr}`) support
- Bracket matching and auto-closing
- Code folding
- Comment toggling (`Ctrl+/` / `Cmd+/`)

## Installation

### From source

```bash
# Symlink into VS Code extensions directory
ln -s $(pwd)/editors/vscode ~/.vscode/extensions/vyr
```

Then reload VS Code. All `.vyr` files will be highlighted automatically.

### Manual

Copy the `editors/vscode` directory to `~/.vscode/extensions/vyr`.
