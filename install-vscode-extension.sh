#!/bin/bash
# Shell script to install Kottg VSCode extension
# Run this script from the kottg project root directory

echo "╔══════════════════════════════════════════════════════════╗"
echo "║     Kottg VSCode Extension Installer (Unix/Mac)          ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# Determine VSCode extensions directory based on OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    EXTENSIONS_DIR="$HOME/.vscode/extensions"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    EXTENSIONS_DIR="$HOME/.vscode/extensions"
else
    echo "❌ Unsupported operating system"
    exit 1
fi

TARGET_DIR="$EXTENSIONS_DIR/kottg-language-1.0.0"

echo "📂 VSCode Extensions Directory: $EXTENSIONS_DIR"
echo ""

# Check if VSCode extensions directory exists
if [ ! -d "$EXTENSIONS_DIR" ]; then
    echo "❌ VSCode extensions directory not found!"
    echo "   Make sure VSCode is installed."
    exit 1
fi

# Remove old installation if exists
if [ -d "$TARGET_DIR" ]; then
    echo "🗑️  Removing old installation..."
    rm -rf "$TARGET_DIR"
fi

# Copy extension
echo "📦 Installing Kottg language extension..."
cp -r vscode-kottg "$TARGET_DIR"

echo ""
echo "✅ Extension installed successfully!"
echo ""
echo "📝 Next steps:"
echo "   1. Restart Visual Studio Code"
echo "   2. Open a .kg file"
echo "   3. Enjoy syntax highlighting and snippets!"
echo ""
echo "🎯 Quick start:"
echo "   - Type 'contract' and press Tab for a contract template"
echo "   - Press Ctrl+Shift+B to compile current contract"
echo "   - Type 'pubfn' and press Tab for a public function"
echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║              Installation Complete! 🎉                   ║"
echo "╚══════════════════════════════════════════════════════════╝"
