# Kottg - KodeChain Smart Contract Compiler

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)

**Kottg** (KodeChain On-chain Transaction Typing & Governance) is a statically-typed, contract-oriented programming language designed specifically for the KodeChain blockchain. It compiles to KodeChain VM bytecode and provides native support for post-quantum cryptography and staking operations.

## Features

✨ **Statically Typed** - Catch errors at compile-time with a robust type system
🔒 **Security First** - Built-in protections against common vulnerabilities
⚡ **Gas-Aware** - Explicit gas costs for all operations
🎯 **Staking-Native** - First-class support for validator and delegation operations
🔍 **Semantic Analysis** - Comprehensive type checking and scope resolution
💎 **Bytecode Generation** - Direct compilation to KodeChain VM bytecode

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [VSCode Integration](#vscode-integration)
- [Compiler Architecture](#compiler-architecture)
- [Language Features](#language-features)
- [Usage](#usage)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/kodechain/kottg.git
cd kottg

# Build the compiler
go build -o kottg main.go

# Verify installation
./kottg -version
```

### Install Globally

```bash
# Install to $GOPATH/bin
go install

# Verify
kottg -version
```

## Quick Start

### 1. Write Your First Contract

Create a file `HelloWorld.kg`:

```kottg
contract HelloWorld {
    state:
        message: string
        owner: address

    init(initialMessage: string) {
        self.message = initialMessage
        self.owner = msg.sender
    }

    pub fn setMessage(newMessage: string) {
        require(msg.sender == self.owner, "Only owner can set message")
        self.message = newMessage
    }

    view fn getMessage() -> string {
        return self.message
    }
}
```

### 2. Compile the Contract

```bash
# Basic compilation
kottg -input HelloWorld.kg

# With verbose output
kottg -input HelloWorld.kg -verbose

# With disassembly
kottg -input HelloWorld.kg -disasm

# Specify output file
kottg -input HelloWorld.kg -output build/HelloWorld.bin
```

### 3. Deploy to KodeChain

The compiler generates bytecode in `.bin` format that can be deployed directly to the KodeChain blockchain.

```bash
# The bytecode is ready for deployment
ls -lh HelloWorld.bin
```

## VSCode Integration

Kottg viene con soporte completo para Visual Studio Code, incluyendo:

- ✨ **Syntax Highlighting** - Resaltado de sintaxis completo
- 🎯 **30+ Snippets** - Templates para código rápido
- 🔧 **Build Tasks** - Compilación integrada con `Ctrl+Shift+B`
- 🐛 **Debug Support** - Debugging del compilador
- 📝 **IntelliSense** - Autocompletado y sugerencias

### Instalación de la Extensión

```bash
# Windows
.\install-extension.ps1

# macOS / Linux
./install-vscode-extension.sh

# Luego reinicia VSCode
```

### Uso Rápido

1. Abre un archivo `.kg`
2. Escribe `contract` y presiona `Tab` para un template
3. Presiona `Ctrl+Shift+B` para compilar
4. ¡Disfruta del syntax highlighting y snippets!

**Ver guía completa**: [VSCODE-SETUP.md](VSCODE-SETUP.md)

### Snippets Disponibles

| Snippet | Descripción |
|---------|-------------|
| `contract` | Plantilla de contrato |
| `pubfn` | Función pública |
| `viewfn` | Función view |
| `require` | Modificador require |
| `emit` | Emitir evento |
| `let` | Declarar variable |
| `assert` | Aserción |

## Compiler Architecture

The Kottg compiler follows a multi-stage compilation pipeline:

```
┌──────────────────┐
│  Source Code     │  .kg file
│  (HelloWorld.kg) │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Lexical Analysis │  Tokenization
│    (Lexer)       │  compiler/lexer.go
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│     Parsing      │  AST Construction
│    (Parser)      │  compiler/parser.go
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Semantic      │  Type Checking
│    Analysis      │  Scope Resolution
│  (Analyzer)      │  compiler/semantic.go
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Bytecode      │  Code Generation
│   Generation     │  KodeChain VM Opcodes
│  (Generator)     │  compiler/bytecode.go
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   Bytecode       │  .bin file
│   Output         │  Ready for deployment
└──────────────────┘
```

### Compilation Stages

#### 1. Lexical Analysis (Lexer)
- **Input**: Source code (.kg file)
- **Output**: Token stream
- **File**: [compiler/lexer.go](compiler/lexer.go)
- **Purpose**: Breaks source code into tokens (keywords, identifiers, operators, etc.)

**Example**:
```kottg
let x: u256 = 42
```
Becomes:
```
TOKEN_LET, TOKEN_IDENT(x), TOKEN_COLON, TOKEN_U256, TOKEN_ASSIGN, TOKEN_INT(42)
```

#### 2. Syntax Analysis (Parser)
- **Input**: Token stream
- **Output**: Abstract Syntax Tree (AST)
- **File**: [compiler/parser.go](compiler/parser.go) + [compiler/ast.go](compiler/ast.go)
- **Purpose**: Validates syntax and builds a tree representation of the program

**AST Nodes**:
- Contracts, Functions, State Variables
- Expressions, Statements
- Type Definitions

#### 3. Semantic Analysis
- **Input**: AST
- **Output**: Validated AST
- **File**: [compiler/semantic.go](compiler/semantic.go)
- **Purpose**: Ensures program correctness

**Checks Performed**:
- ✅ Type checking (type safety)
- ✅ Scope resolution (variable visibility)
- ✅ Function signature validation
- ✅ Return type verification
- ✅ Mutability checks
- ✅ View function constraints
- ✅ Modifier existence
- ✅ Precompiled function calls

**Example Error Detection**:
```kottg
let x: u256 = "hello"  // ❌ Error: type mismatch
let y: bool = x > 10   // ✅ Correct
```

#### 4. Bytecode Generation
- **Input**: Validated AST
- **Output**: KodeChain VM bytecode
- **File**: [compiler/bytecode.go](compiler/bytecode.go)
- **Purpose**: Generates executable bytecode for KodeChain VM

**Generated Opcodes**:
- Standard EVM opcodes (ADD, SUB, SLOAD, SSTORE, etc.)
- Custom KodeChain opcodes:
  - `0xc0` - GETSTAKE
  - `0xc1` - ISVALIDATOR
  - `0xc2` - DISTRIBUTEFEES
  - `0xc3` - GETVALIDATORINFO

## Language Features

### Data Types

#### Primitive Types
```kottg
bool           // true/false
u8 to u256     // Unsigned integers
i8 to i256     // Signed integers
address        // 20-byte address
hash           // 32-byte hash
bytes          // Dynamic byte array
string         // UTF-8 string
```

#### Complex Types
```kottg
vec[T]         // Dynamic vector
map[K, V]      // Hash map
struct         // Custom structs
```

### Contract Structure

```kottg
contract MyContract {
    // State variables
    state:
        owner: address
        balance: u256

    // Events
    event Transfer {
        from: address indexed,
        to: address indexed,
        amount: u256
    }

    // Constructor
    init(ownerAddr: address) {
        self.owner = ownerAddr
        self.balance = 0
    }

    // Modifiers
    require fn onlyOwner() {
        assert(msg.sender == self.owner, "Not owner")
    }

    // Public function
    pub fn transfer(to: address, amount: u256) onlyOwner {
        require(amount <= self.balance, "Insufficient balance")
        self.balance -= amount
        emit Transfer { from: self.owner, to: to, amount: amount }
    }

    // View function (read-only)
    view fn getBalance() -> u256 {
        return self.balance
    }
}
```

### Built-in Globals

```kottg
// Message context
msg.sender      // Current caller
msg.value       // Sent value
msg.gas         // Remaining gas

// Block context
block.number    // Current block number
block.timestamp // Block timestamp
block.validator // Block validator

// Transaction context
tx.origin       // Original sender
tx.gasprice     // Gas price

// Contract context
self.address    // Contract address
self.balance    // Contract balance
```

### Precompiled Contracts

Kottg provides native access to KodeChain staking operations:

```kottg
// Get validator stake
let stake = precompiled.getStake(validator_address)

// Check if address is a validator
let isValidator = precompiled.isValidator(address)

// Distribute fees to validator
precompiled.distributeFees(validator, amount)

// Get complete validator info
let info = precompiled.getValidatorInfo(validator)
```

### Control Flow

```kottg
// If-else
if condition {
    // ...
} else if other {
    // ...
} else {
    // ...
}

// Loops
loop i in 0..10 {
    // ...
}

while condition {
    // ...
}

// Match (pattern matching)
match value {
    0 => { /* ... */ },
    1..10 => { /* ... */ },
    _ => { /* default */ }
}
```

## Usage

### Command-Line Options

```bash
kottg [options]

Options:
  -input string
        Input kottg source file (.kg)
  -output string
        Output bytecode file (default: <input>.bin)
  -disasm
        Show disassembled bytecode
  -ast
        Show Abstract Syntax Tree
  -verbose
        Verbose output
  -json
        Output bytecode metadata as JSON
  -version
        Show version information

Examples:
  kottg -input contracts/StakingQuery.kg
  kottg -input contracts/StakingQuery.kg -disasm -verbose
  kottg -input contracts/Token.kg -output build/Token.bin
  kottg -input contracts/MyContract.kg -ast -json
```

### Compilation Workflow

```bash
# 1. Write your contract
vim contracts/MyContract.kg

# 2. Compile with full output
kottg -input contracts/MyContract.kg -verbose -disasm

# 3. Check the generated bytecode
xxd MyContract.bin | head

# 4. Deploy to KodeChain
# (Use KodeChain CLI or Web3 tools)
```

## Examples

### Example 1: Simple Storage

```kottg
contract SimpleStorage {
    state:
        storedData: u256

    init() {
        self.storedData = 0
    }

    pub fn set(x: u256) {
        self.storedData = x
    }

    view fn get() -> u256 {
        return self.storedData
    }
}
```

### Example 2: Staking Query Contract

See [contracts/StakingQuery.kg](contracts/StakingQuery.kg) for a complete example using precompiled staking operations.

```kottg
contract StakingQuery {
    state:
        owner: address
        totalQueries: u64
        minStake: u256

    event StakeQueried {
        validator: address indexed,
        stake: u256,
        timestamp: u64
    }

    init(ownerAddr: address, minimumStake: u256) {
        self.owner = ownerAddr
        self.totalQueries = 0
        self.minStake = minimumStake
    }

    pub fn getValidatorStake(validator: address) -> u256 {
        self.totalQueries += 1
        let stake = precompiled.getStake(validator)

        emit StakeQueried {
            validator: validator,
            stake: stake,
            timestamp: block.timestamp
        }

        return stake
    }

    pub fn isValidatorActive(validator: address) -> bool {
        let stake = precompiled.getStake(validator)
        let isActive = precompiled.isValidator(validator)
        return isActive && stake >= self.minStake
    }

    view fn getQueryCount() -> u64 {
        return self.totalQueries
    }
}
```

### Running the Examples

```bash
# Compile the staking query contract
kottg -input contracts/StakingQuery.kg -verbose

# View the disassembly
kottg -input contracts/StakingQuery.kg -disasm

# Generate JSON metadata
kottg -input contracts/StakingQuery.kg -json
```

## Development

### Project Structure

```
kottg/
├── compiler/           # Compiler implementation
│   ├── lexer.go       # Lexical analyzer
│   ├── parser.go      # Syntax parser
│   ├── ast.go         # AST definitions
│   ├── semantic.go    # Semantic analyzer
│   └── bytecode.go    # Bytecode generator
├── contracts/         # Example contracts
│   └── StakingQuery.kg
├── examples/          # Demo programs
│   ├── lexer_demo.go
│   └── parser_demo.go
├── main.go           # Compiler entry point
├── go.mod            # Go module definition
├── SPEC.md           # Language specification
└── README.md         # This file
```

### Running Tests

```bash
# Run all tests
go test ./...

# Test specific package
go test ./compiler

# With coverage
go test -cover ./...
```

### Building

```bash
# Build for current platform
go build -o kottg main.go

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o kottg-linux-amd64
GOOS=windows GOARCH=amd64 go build -o kottg-windows-amd64.exe
GOOS=darwin GOARCH=arm64 go build -o kottg-darwin-arm64
```

### Code Style

This project follows standard Go conventions:
- Use `gofmt` to format code
- Run `go vet` to check for common errors
- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines

## Semantic Analysis Details

The semantic analyzer performs comprehensive validation:

### Type Checking

```kottg
// ✅ Valid: matching types
let x: u256 = 42
let y: u256 = x + 10

// ❌ Error: type mismatch
let z: u256 = "hello"

// ✅ Valid: boolean expression
if x > 10 {
    // ...
}

// ❌ Error: non-boolean condition
if x {  // Error: condition must be boolean
    // ...
}
```

### Scope Resolution

```kottg
contract Example {
    state:
        count: u64

    pub fn increment() {
        let temp: u64 = 1
        self.count += temp  // ✅ Valid: temp is in scope
    }

    pub fn test() {
        self.count += temp  // ❌ Error: temp not in scope
    }
}
```

### View Function Validation

```kottg
contract Example {
    state:
        value: u256

    // ✅ Valid: view function, no state modification
    view fn getValue() -> u256 {
        return self.value
    }

    // ❌ Error: view function cannot modify state
    view fn setValue(v: u256) {
        self.value = v  // Error!
    }
}
```

## Bytecode Output

### Disassembly Example

```
═══════════════════════════════════════════
DISASSEMBLY
═══════════════════════════════════════════

0000: PUSH1           0x00
0002: CALLDATALOAD
0003: PUSH1           0x00
0005: SLOAD
0006: PUSH1           0x01
0008: ADD
0009: PUSH1           0x00
0011: SSTORE
0012: PUSH1           0x00
0014: PUSH1           0x00
0016: RETURN
```

### Custom Opcodes

The compiler generates special opcodes for KodeChain operations:

```
0xc0  GETSTAKE        - Get validator stake
0xc1  ISVALIDATOR     - Check if address is validator
0xc2  DISTRIBUTEFEES  - Distribute fees to validator
0xc3  GETVALIDATORINFO- Get complete validator information
```

## Error Messages

The compiler provides detailed error messages:

```
❌ SEMANTIC ERRORS:

1. type mismatch in variable x: declared as u256, assigned string
2. undefined identifier: unknownVar
3. view functions cannot modify state
4. operator + requires numeric operands, got bool and u256
```

## Gas Costs

All operations have explicit gas costs (based on EVM):

- Storage write (SSTORE): 20,000 gas
- Storage read (SLOAD): 200 gas
- Arithmetic operations: 3-5 gas
- Precompiled calls: 500-5,000 gas

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Update documentation
- Follow Go coding standards
- Add examples for new language features

## Roadmap

- [ ] Optimizer pass for bytecode
- [ ] Debugging information generation
- [ ] IDE language server support
- [ ] Standard library expansion
- [ ] Formal verification tools
- [ ] Gas estimation tools

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- KodeChain Core Team
- Ethereum Foundation (EVM inspiration)
- Go programming language community

## Support

- Documentation: See [SPEC.md](SPEC.md)
- Issues: [GitHub Issues](https://github.com/kodechain/kottg/issues)
- Discord: [KodeChain Community](https://discord.gg/kodechain)

---

**Made with ❤️ for the KodeChain ecosystem**

*Version 1.0.0 - Last updated: 2025-11-14*
