# VSCode Setup Guide for Kottg Development

Esta guía te ayudará a configurar Visual Studio Code para desarrollar contratos inteligentes en Kottg.

## 📦 Instalación de la Extensión

### Windows

```powershell
# Desde el directorio del proyecto
.\install-extension.ps1
```

### macOS / Linux

```bash
# Desde el directorio del proyecto
chmod +x install-vscode-extension.sh
./install-vscode-extension.sh
```

### Manual

1. Copia la carpeta `vscode-kottg` a:
   - **Windows**: `%USERPROFILE%\.vscode\extensions\kottg-language-1.0.0`
   - **macOS**: `~/.vscode/extensions/kottg-language-1.0.0`
   - **Linux**: `~/.vscode/extensions/kottg-language-1.0.0`

2. Reinicia VSCode

## ✨ Características

### 1. Syntax Highlighting

La extensión proporciona resaltado de sintaxis completo para:

- **Palabras clave**: `contract`, `state`, `fn`, `pub`, `view`, `require`, etc.
- **Tipos**: `u256`, `address`, `bool`, `string`, `vec`, `map`
- **Built-ins**: `msg.sender`, `block.timestamp`, `self.balance`
- **Operadores**: `+`, `-`, `==`, `&&`, `||`
- **Comentarios**: `//` y `/* */`
- **Strings y números**: Con soporte para hexadecimales

### 2. Code Snippets

Escribe y presiona `Tab` para expandir:

#### Estructuras de Contrato

| Snippet | Descripción |
|---------|-------------|
| `contract` | Plantilla básica de contrato |
| `contractfull` | Plantilla completa con patrones comunes |
| `state` | Bloque de variables de estado |
| `init` | Función constructor |
| `struct` | Definición de estructura |
| `event` | Declaración de evento |

#### Funciones

| Snippet | Descripción |
|---------|-------------|
| `pubfn` | Función pública |
| `viewfn` | Función view (solo lectura) |
| `require` | Modificador require |
| `onlyowner` | Modificador onlyOwner |

#### Statements

| Snippet | Descripción |
|---------|-------------|
| `let` | Declaración de variable |
| `letmut` | Variable mutable |
| `if` | Sentencia if |
| `ifelse` | Sentencia if-else |
| `loop` | Bucle for |
| `while` | Bucle while |
| `match` | Expresión match |
| `emit` | Emitir evento |

#### Validación

| Snippet | Descripción |
|---------|-------------|
| `assert` | Aserción |
| `req` | Require |
| `revert` | Revert |

#### Precompilados

| Snippet | Descripción |
|---------|-------------|
| `getstake` | Obtener stake de validador |
| `isvalidator` | Verificar si es validador |

### 3. Tareas de Compilación

Presiona `Ctrl+Shift+B` para acceder a las tareas:

1. **Compile Current Contract** - Compila el archivo actual (tarea por defecto)
2. **Build Kottg Compiler** - Compila el compilador
3. **Compile Contract with AST** - Muestra el AST
4. **Compile Contract with Disassembly** - Muestra el bytecode
5. **Compile Contract (All Options)** - Todas las opciones de debug
6. **Compile All Example Contracts** - Compila todos los ejemplos

### 4. Debugging

Presiona `F5` para debuggear:

- **Debug Kottg Compiler** - Debug del compilador con StakingQuery.kg
- **Debug Current Contract** - Debug con el contrato actual

## 🎯 Workflow de Desarrollo

### 1. Crear un Nuevo Contrato

1. Crea un archivo `MiContrato.kg`
2. Escribe `contract` y presiona `Tab`
3. Completa el template

```kottg
contract MiContrato {
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

### 2. Compilar

- **Atajo rápido**: `Ctrl+Shift+B` → "Compile Current Contract"
- **Terminal**: `./kottg.exe -input MiContrato.kg -verbose`

### 3. Ver el Bytecode

```bash
# Ver desensamblado
./kottg.exe -input MiContrato.kg -disasm

# Ver AST
./kottg.exe -input MiContrato.kg -ast

# Todo junto
./kottg.exe -input MiContrato.kg -verbose -ast -disasm -json
```

### 4. Debuggear el Compilador

1. Abre `main.go`
2. Establece breakpoints
3. Presiona `F5`
4. Selecciona "Debug Current Contract"

## ⌨️ Atajos de Teclado

| Atajo | Acción |
|-------|--------|
| `Ctrl+Shift+B` | Compilar contrato actual |
| `F5` | Iniciar debugging |
| `Ctrl+K Ctrl+C` | Comentar línea |
| `Ctrl+K Ctrl+U` | Descomentar línea |
| `Ctrl+/` | Toggle comentario |
| `Alt+Up/Down` | Mover línea |
| `Shift+Alt+Down` | Duplicar línea |

## 🔧 Configuración Personalizada

### settings.json

Agrega configuración personalizada en `.vscode/settings.json`:

```json
{
    "files.associations": {
        "*.kg": "kottg"
    },
    "[kottg]": {
        "editor.tabSize": 4,
        "editor.insertSpaces": true,
        "editor.wordWrap": "on",
        "editor.formatOnSave": false
    }
}
```

### tasks.json

Personaliza tareas en `.vscode/tasks.json`:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "My Custom Task",
            "type": "shell",
            "command": "./kottg.exe",
            "args": ["-input", "${file}", "-verbose"],
            "group": "build"
        }
    ]
}
```

## 📝 Ejemplo Completo de Desarrollo

### Paso 1: Crear contrato con snippet

1. Escribe `contractfull` y presiona Tab
2. Completa los placeholders:
   - Nombre del contrato
   - Variables de estado
   - Eventos
   - Funciones

### Paso 2: Agregar función con snippet

1. Escribe `pubfn` y presiona Tab
2. Define:
   - Nombre de función
   - Parámetros
   - Tipo de retorno

### Paso 3: Agregar validación

1. Escribe `assert` y presiona Tab
2. Define condición y mensaje

### Paso 4: Compilar y verificar

1. `Ctrl+Shift+B`
2. Verifica que no hay errores
3. Revisa el bytecode generado

## 🎨 Temas de Color

La extensión funciona con cualquier tema de VSCode. Recomendados:

- **Dark+** (default)
- **Monokai**
- **Dracula**
- **Nord**
- **One Dark Pro**

## 🐛 Solución de Problemas

### La extensión no se carga

1. Verifica que está en el directorio correcto:
   ```bash
   ls ~/.vscode/extensions/kottg-language-1.0.0
   ```

2. Reinicia VSCode completamente

3. Verifica los logs: `View` → `Output` → `Log (Extension Host)`

### Syntax highlighting no funciona

1. Verifica que el archivo tiene extensión `.kg`
2. Abre `View` → `Command Palette` → "Change Language Mode"
3. Selecciona "Kottg"

### Tasks no funcionan

1. Verifica que `kottg.exe` existe en la raíz del proyecto
2. Compila el compilador:
   ```bash
   go build -o kottg.exe main.go
   ```

### Snippets no se expanden

1. Escribe el prefijo completo
2. Presiona `Tab` (no Enter)
3. Verifica que IntelliSense está habilitado

## 📚 Recursos

- [Especificación del Lenguaje](SPEC.md)
- [Guía Rápida](QUICKSTART.md)
- [README Principal](README.md)
- [Ejemplos](contracts/)

## 🚀 Próximos Pasos

1. **Explorar ejemplos**: Abre `contracts/StakingQuery.kg`
2. **Probar snippets**: Escribe `contract` + Tab
3. **Compilar**: Presiona `Ctrl+Shift+B`
4. **Crear tu contrato**: Usa los templates

## 💡 Tips y Trucos

### IntelliSense

- `Ctrl+Space` para activar autocompletado
- Las sugerencias aparecen automáticamente al escribir

### Multi-cursor

- `Alt+Click` para agregar cursor
- `Ctrl+Alt+Down/Up` para cursor en líneas adyacentes

### Búsqueda y reemplazo

- `Ctrl+F` para buscar
- `Ctrl+H` para buscar y reemplazar
- `Ctrl+Shift+F` para buscar en todos los archivos

### Navegación

- `Ctrl+P` para abrir archivo rápido
- `Ctrl+G` para ir a línea
- `Ctrl+Shift+O` para ir a símbolo

---

¡Disfruta desarrollando contratos inteligentes con Kottg! 🎉
