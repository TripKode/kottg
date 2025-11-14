# Kottg Language Support for Visual Studio Code

Official Visual Studio Code extension for the Kottg programming language - KodeChain's smart contract language.

## Features

- **Syntax Highlighting** - Full syntax highlighting for Kottg language
- **Code Snippets** - Quick snippets for common patterns
- **Auto-completion** - IntelliSense support
- **Bracket Matching** - Automatic bracket and quote pairing
- **Code Folding** - Fold/unfold code blocks
- **Comment Support** - Line and block comments

## Installation

### From Source

1. Copy the `vscode-kottg` folder to your VSCode extensions directory:
   - **Windows**: `%USERPROFILE%\.vscode\extensions\`
   - **macOS**: `~/.vscode/extensions/`
   - **Linux**: `~/.vscode/extensions/`

2. Restart VSCode

### Quick Setup

```bash
# Windows
cp -r vscode-kottg %USERPROFILE%\.vscode\extensions\kottg-language-1.0.0

# macOS/Linux
cp -r vscode-kottg ~/.vscode/extensions/kottg-language-1.0.0
```

## Usage

### Compile Current File

1. Open a `.kg` file
2. Press `Ctrl+Shift+B` (or `Cmd+Shift+B` on macOS)
3. Select "Compile Current Contract"

### Available Tasks

- **Build Kottg Compiler** - Compile the compiler itself
- **Compile Current Contract** - Compile the currently open .kg file
- **Compile Contract with AST** - Compile and show AST
- **Compile Contract with Disassembly** - Compile and show bytecode
- **Compile Contract (All Options)** - Compile with all debug options
- **Compile All Example Contracts** - Compile all contracts in contracts/

### Keyboard Shortcuts

- `Ctrl+Shift+B` - Build/Compile current contract
- `F5` - Debug compiler with current contract

## Snippets

Type these prefixes and press `Tab`:

### Contract Structure

- `contract` - Basic contract template
- `contractfull` - Full contract with common patterns
- `state` - State variables block
- `init` - Constructor function
- `struct` - Struct definition
- `event` - Event definition

### Functions

- `pubfn` - Public function
- `viewfn` - View function (read-only)
- `require` - Require modifier
- `onlyowner` - Only owner modifier template

### Statements

- `let` - Variable declaration
- `letmut` - Mutable variable
- `if` - If statement
- `ifelse` - If-else statement
- `loop` - Loop statement
- `while` - While loop
- `match` - Match statement
- `emit` - Emit event

### Validation

- `assert` - Assert statement
- `req` - Require statement
- `revert` - Revert statement

### Precompiled Functions

- `getstake` - Get validator stake
- `isvalidator` - Check if validator

## Syntax Highlighting

The extension provides comprehensive syntax highlighting for:

- **Keywords**: `contract`, `state`, `fn`, `pub`, `view`, etc.
- **Types**: `u256`, `address`, `bool`, `string`, etc.
- **Built-ins**: `msg`, `block`, `tx`, `precompiled`
- **Operators**: `+`, `-`, `==`, `&&`, etc.
- **Comments**: `//` and `/* */`
- **Strings**: `"..."` with escape sequences
- **Numbers**: Decimal and hexadecimal

## Example

Create a new file `MyContract.kg` and start typing:

```kottg
contract MyContract {
    state:
        owner: address
        balance: u256

    init(ownerAddr: address) {
        self.owner = ownerAddr
        self.balance = 0
    }

    pub fn deposit() {
        self.balance += msg.value
    }

    view fn getBalance() -> u256 {
        return self.balance
    }
}
```

## Configuration

The extension works with the workspace settings in `.vscode/settings.json`:

```json
{
    "files.associations": {
        "*.kg": "kottg"
    },
    "[kottg]": {
        "editor.tabSize": 4,
        "editor.insertSpaces": true
    }
}
```

## Troubleshooting

### Extension not working

1. Ensure the extension is in the correct folder
2. Restart VSCode
3. Check that `.kg` files are associated with "kottg" language

### Compilation errors

1. Ensure `kottg.exe` is built (`go build main.go`)
2. Check that the compiler is in the workspace root
3. Verify the contract syntax is correct

## Contributing

Contributions are welcome! Please see the main repository for guidelines.

## License

MIT License - see LICENSE file

## Links

- [Kottg Repository](https://github.com/kodechain/kottg)
- [Language Specification](../SPEC.md)
- [Quick Start Guide](../QUICKSTART.md)

---

Made with ❤️ for the KodeChain ecosystem
