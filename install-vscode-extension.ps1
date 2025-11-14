# PowerShell script to install Kottg VSCode extension
# Run this script from the kottg project root directory

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "     Kottg VSCode Extension Installer (Windows)          " -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# Get VSCode extensions directory
$extensionsDir = "$env:USERPROFILE\.vscode\extensions"
$targetDir = "$extensionsDir\kottg-language-1.0.0"

Write-Host "📂 VSCode Extensions Directory: $extensionsDir" -ForegroundColor Yellow
Write-Host ""

# Check if VSCode extensions directory exists
if (-Not (Test-Path $extensionsDir)) {
    Write-Host "❌ VSCode extensions directory not found!" -ForegroundColor Red
    Write-Host "   Make sure VSCode is installed." -ForegroundColor Red
    exit 1
}

# Remove old installation if exists
if (Test-Path $targetDir) {
    Write-Host "🗑️  Removing old installation..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $targetDir
}

# Copy extension
Write-Host "📦 Installing Kottg language extension..." -ForegroundColor Green
Copy-Item -Recurse -Force "vscode-kottg" $targetDir

Write-Host ""
Write-Host "✅ Extension installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 Next steps:" -ForegroundColor Cyan
Write-Host "   1. Restart Visual Studio Code" -ForegroundColor White
Write-Host "   2. Open a .kg file" -ForegroundColor White
Write-Host "   3. Enjoy syntax highlighting and snippets!" -ForegroundColor White
Write-Host ""
Write-Host "🎯 Quick start:" -ForegroundColor Cyan
Write-Host "   - Type 'contract' and press Tab for a contract template" -ForegroundColor White
Write-Host "   - Press Ctrl+Shift+B to compile current contract" -ForegroundColor White
Write-Host "   - Type 'pubfn' and press Tab for a public function" -ForegroundColor White
Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "              Installation Complete!                      " -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
