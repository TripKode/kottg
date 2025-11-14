# Guía Rápida de Inicio - Kottg Compiler

## Instalación y Construcción

```bash
# 1. Clonar el repositorio
git clone https://github.com/kodechain/kottg.git
cd kottg

# 2. Compilar el compilador
go build -o kottg main.go

# En Windows
go build -o kottg.exe main.go

# 3. Verificar instalación
./kottg -version
```

## Compilar tu Primer Contrato

### Opción 1: Usar un contrato de ejemplo

```bash
# Compilación básica
./kottg -input contracts/StakingQuery.kg

# Con salida detallada
./kottg -input contracts/StakingQuery.kg -verbose

# Ver el AST
./kottg -input contracts/StakingQuery.kg -ast

# Ver el bytecode desensamblado
./kottg -input contracts/StakingQuery.kg -disasm

# Exportar metadata en JSON
./kottg -input contracts/StakingQuery.kg -json

# Todas las opciones juntas
./kottg -input contracts/StakingQuery.kg -verbose -ast -disasm -json
```

### Opción 2: Crear tu propio contrato

Crea un archivo `MiContrato.kg`:

```kottg
contract MiContrato {
    state:
        contador: u256
        propietario: address

    event Incrementado {
        nuevoValor: u256
    }

    init(valorInicial: u256) {
        self.contador = valorInicial
        self.propietario = msg.sender
    }

    pub fn incrementar() {
        self.contador += 1
        emit Incrementado {
            nuevoValor: self.contador
        }
    }

    view fn obtenerContador() -> u256 {
        return self.contador
    }
}
```

Luego compílalo:

```bash
./kottg -input MiContrato.kg -verbose
```

## Salida del Compilador

El compilador genera:

1. **Archivo .bin** - Bytecode listo para desplegar
2. **Mensajes de error** - Si hay problemas de sintaxis o semántica
3. **Información de depuración** - Con las opciones -verbose, -ast, -disasm

### Ejemplo de salida exitosa:

```
═══════════════════════════════════════════
KOTTG COMPILER v1.0.0
═══════════════════════════════════════════

Input file: MiContrato.kg
Source size: 456 bytes

Phase 1: Lexical Analysis...
Phase 2: Parsing...
✅ Parsing successful!
   Contracts: 1
   Top-level structs: 0

Phase 3: Semantic Analysis...
✅ Semantic analysis successful!

Phase 4: Bytecode Generation...
✅ Bytecode generation successful!
   Bytecode size: 85 bytes


═══════════════════════════════════════════
✅ COMPILATION SUCCESSFUL
═══════════════════════════════════════════

Contract compiled successfully!
Output: MiContrato.bin
Bytecode: 0x600055600060015560025560003560e0...
```

## Errores Comunes y Soluciones

### Error: Type mismatch

```
❌ SEMANTIC ERRORS:
1. type mismatch in variable x: declared as u256, assigned string
```

**Solución**: Asegúrate de que los tipos coincidan:

```kottg
// ❌ Incorrecto
let x: u256 = "hello"

// ✅ Correcto
let x: u256 = 42
let y: string = "hello"
```

### Error: Undefined identifier

```
❌ SEMANTIC ERRORS:
1. undefined identifier: miVariable
```

**Solución**: Declara la variable antes de usarla:

```kottg
// ❌ Incorrecto
return miVariable

// ✅ Correcto
let miVariable: u256 = 100
return miVariable
```

### Error: View functions cannot modify state

```
❌ SEMANTIC ERRORS:
1. view functions cannot modify state
```

**Solución**: Elimina el modificador `view` si modificas el estado:

```kottg
// ❌ Incorrecto
view fn cambiarEstado() {
    self.contador = 10  // Error!
}

// ✅ Correcto
pub fn cambiarEstado() {
    self.contador = 10
}
```

## Características Principales

### 1. Tipos de Datos

```kottg
// Enteros sin signo
let a: u256 = 1000
let b: u64 = 100
let c: u8 = 255

// Address
let addr: address = msg.sender

// Boolean
let activo: bool = true

// String
let mensaje: string = "Hola KodeChain"

// Vectores
let numeros: vec[u256] = vec[1, 2, 3]

// Mapas
let balances: map[address, u256]
```

### 2. Contextos Globales

```kottg
// Información del mensaje
msg.sender      // Quien llamó a la función
msg.value       // Valor enviado
msg.gas         // Gas restante

// Información del bloque
block.number    // Número de bloque actual
block.timestamp // Timestamp del bloque
block.validator // Validador del bloque

// Información de la transacción
tx.origin       // Origen de la transacción
tx.gasprice     // Precio del gas
```

### 3. Funciones Precompiladas de KodeChain

```kottg
// Obtener stake de un validador
let stake = precompiled.getStake(validatorAddr)

// Verificar si es validador
let esValidador = precompiled.isValidator(validatorAddr)

// Distribuir fees
precompiled.distributeFees(validatorAddr, cantidad)

// Obtener información completa del validador
let info = precompiled.getValidatorInfo(validatorAddr)
```

### 4. Modificadores y Permisos

```kottg
// Definir modificador
require fn soloPropietario() {
    assert(msg.sender == self.propietario, "No autorizado")
}

// Usar modificador
pub fn funcionProtegida() soloPropietario {
    // Solo el propietario puede ejecutar esto
}
```

### 5. Eventos

```kottg
// Declarar evento
event Transferencia {
    desde: address indexed,
    hacia: address indexed,
    cantidad: u256
}

// Emitir evento
emit Transferencia {
    desde: msg.sender,
    hacia: destinatario,
    cantidad: monto
}
```

## Siguientes Pasos

1. **Experimenta con los contratos de ejemplo**: `contracts/StakingQuery.kg` y `contracts/SimpleStorage.kg`

2. **Lee la especificación completa**: [SPEC.md](SPEC.md)

3. **Explora el README**: [README.md](README.md) para documentación detallada

4. **Prueba todas las opciones del compilador**:
   ```bash
   ./kottg -input contracts/StakingQuery.kg -verbose -ast -disasm -json
   ```

5. **Crea tus propios contratos** y compílalos para KodeChain

## Recursos Adicionales

- **Especificación del Lenguaje**: [SPEC.md](SPEC.md)
- **Documentación Completa**: [README.md](README.md)
- **Código Fuente**: [compiler/](compiler/)
- **Ejemplos**: [contracts/](contracts/)

## Ayuda y Soporte

Si encuentras problemas:

1. Verifica que el archivo `.kg` tenga sintaxis correcta
2. Usa `-verbose` para ver información detallada
3. Revisa los ejemplos en `contracts/`
4. Consulta la especificación en `SPEC.md`

---

¡Feliz compilación! 🚀
