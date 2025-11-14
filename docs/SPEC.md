# Kottg Language Specification v1.0

## Overview

**kottg** (KodeChain On-chain Transaction Typing & Governance) is a statically-typed, contract-oriented programming language designed specifically for the KodeChain blockchain. It compiles to KodeChain VM bytecode and provides native support for post-quantum cryptography and staking operations.

## Philosophy

- **Simple but not simplistic**: Easy to learn, powerful to use
- **Security first**: Built-in protections against common vulnerabilities
- **Gas-aware**: Explicit gas costs for operations
- **Staking-native**: First-class support for validator and delegation operations

## Basic Syntax

### Contract Declaration

```kottg
contract StakingQuery {
    // State variables
    state:
        owner: address
        totalQueries: u64

    // Constructor
    init(ownerAddr: address) {
        self.owner = ownerAddr
        self.totalQueries = 0
    }

    // Public function
    pub fn getValidatorStake(validator: address) -> u256 {
        self.totalQueries += 1
        return precompiled.getStake(validator)
    }

    // View function (no state changes)
    view fn getQueryCount() -> u64 {
        return self.totalQueries
    }

    // Permission check
    require fn onlyOwner() {
        assert(msg.sender == self.owner, "Not owner")
    }
}
```

### Data Types

#### Primitive Types
- `bool` - Boolean (true/false)
- `u8`, `u16`, `u32`, `u64`, `u128`, `u256` - Unsigned integers
- `i8`, `i16`, `i32`, `i64`, `i128`, `i256` - Signed integers
- `address` - 20-byte Ethereum-compatible address
- `hash` - 32-byte hash value
- `bytes` - Dynamic byte array
- `string` - UTF-8 string

#### Complex Types
- `array[T, N]` - Fixed-size array
- `vec[T]` - Dynamic vector
- `map[K, V]` - Hash map

### Built-in Globals

```kottg
// Message context
msg.sender: address      // Transaction sender
msg.value: u256         // Sent value in KDC
msg.data: bytes         // Transaction data
msg.gas: u64            // Remaining gas

// Block context
block.number: u64       // Current block number
block.timestamp: u64    // Block timestamp
block.validator: address // Block validator

// Transaction context
tx.origin: address      // Original transaction sender
tx.gasprice: u256      // Gas price

// Contract context
self.address: address   // This contract's address
self.balance: u256     // This contract's balance
```

### Precompiled Contracts

kottg provides native access to KodeChain precompiled contracts:

```kottg
// Staking operations (opcodes 0xc0-0xc3)
precompiled.getStake(validator: address) -> u256
precompiled.isValidator(validator: address) -> bool
precompiled.distributeFees(validator: address, amount: u256) -> bool
precompiled.getValidatorInfo(validator: address) -> ValidatorInfo

// ValidatorInfo structure
struct ValidatorInfo {
    stake: u256,
    isActive: bool,
    totalFees: u256,
    blocksValidated: u256
}
```

### Control Flow

```kottg
// If-else
if condition {
    // ...
} else if other_condition {
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
    _ => { /* ... */ } // default
}
```

### Error Handling

```kottg
// Assert with custom message
assert(condition, "Error message")

// Require (reverts if false)
require(balance >= amount, "Insufficient balance")

// Revert with message
revert("Operation failed")

// Try-catch for external calls
try external_contract.method(args) {
    // Success path
} catch (error) {
    // Error handling
}
```

### Events

```kottg
// Event declaration
event StakeQueried {
    validator: address indexed,
    stake: u256,
    timestamp: u64
}

// Emit event
emit StakeQueried {
    validator: validator_addr,
    stake: stake_amount,
    timestamp: block.timestamp
}
```

### Modifiers/Permissions

```kottg
// Permission function
require fn onlyOwner() {
    assert(msg.sender == self.owner, "Not owner")
}

require fn validAmount(amount: u256) {
    assert(amount > 0, "Amount must be positive")
}

// Usage
pub fn withdraw(amount: u256) onlyOwner validAmount(amount) {
    // Function body
}
```

### Complete Example

```kottg
contract ValidatorManager {
    state:
        owner: address
        minStake: u256
        validators: map[address, bool]
        totalValidators: u64

    event ValidatorAdded {
        validator: address indexed,
        stake: u256
    }

    event ValidatorRemoved {
        validator: address indexed
    }

    init(ownerAddr: address, minimumStake: u256) {
        self.owner = ownerAddr
        self.minStake = minimumStake
        self.totalValidators = 0
    }

    require fn onlyOwner() {
        assert(msg.sender == self.owner, "Not owner")
    }

    pub fn addValidator(validator: address) onlyOwner {
        require(!self.validators[validator], "Already exists")

        let stake = precompiled.getStake(validator)
        require(stake >= self.minStake, "Insufficient stake")

        self.validators[validator] = true
        self.totalValidators += 1

        emit ValidatorAdded {
            validator: validator,
            stake: stake
        }
    }

    pub fn removeValidator(validator: address) onlyOwner {
        require(self.validators[validator], "Not found")

        self.validators[validator] = false
        self.totalValidators -= 1

        emit ValidatorRemoved {
            validator: validator
        }
    }

    view fn isValidator(validator: address) -> bool {
        return self.validators[validator] && precompiled.isValidator(validator)
    }

    view fn getValidatorCount() -> u64 {
        return self.totalValidators
    }

    view fn getValidatorStake(validator: address) -> u256 {
        require(self.validators[validator], "Not a registered validator")
        return precompiled.getStake(validator)
    }
}
```

## Compilation

kottg compiles to KodeChain VM bytecode through these stages:

1. **Lexical Analysis**: Source code → Tokens
2. **Parsing**: Tokens → Abstract Syntax Tree (AST)
3. **Semantic Analysis**: Type checking, variable resolution
4. **Optimization**: Constant folding, dead code elimination
5. **Code Generation**: AST → Bytecode

## Gas Costs

All operations have explicit gas costs:
- Storage write: 20,000 gas
- Storage read: 200 gas
- Arithmetic: 3-5 gas
- Precompiled calls: 500-5,000 gas (see VM spec)

## Safety Features

1. **No integer overflow**: All arithmetic is checked
2. **No null pointers**: All values must be initialized
3. **Bounds checking**: Array access is always checked
4. **Reentrancy guard**: Built-in protection
5. **Gas limits**: Prevent infinite loops

## Standard Library

```kottg
// Math operations
math.min(a, b)
math.max(a, b)
math.abs(x)
math.pow(base, exp)

// Cryptographic functions
crypto.hash256(data: bytes) -> hash
crypto.verify_signature(hash, signature, pubkey) -> bool

// String operations
string.concat(s1, s2) -> string
string.length(s) -> u64

// Array operations
array.length(arr) -> u64
array.push(arr, item)
array.pop(arr) -> T
```

## Reserved Keywords

```
contract, state, init, fn, pub, view, require, assert, revert, emit, event
if, else, match, loop, while, in, return, let, mut, self, msg, block, tx
precompiled, true, false, bool, address, hash, bytes, string, struct, enum
```

## File Structure

```
MyContract.kg          # Main contract file
├── contract MyContract
├── dependencies/      # Import path
└── tests/            # Test files
```

## Comments

```kottg
// Single line comment

/*
   Multi-line
   comment
*/

/// Documentation comment for functions/contracts
```

---

**Version**: 1.0.0
**Authors**: KodeChain Core Team
**License**: MIT
**Last Updated**: 2025-11-14
