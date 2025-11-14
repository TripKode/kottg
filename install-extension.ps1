# Kottg VSCode Extension Installer for Windows

$extensionsDir = "$env:USERPROFILE\.vscode\extensions"
$targetDir = "$extensionsDir\kottg-language-1.0.0"

Write-Host ""
Write-Host "Kottg VSCode Extension Installer" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""

if (-Not (Test-Path $extensionsDir)) {
    Write-Host "ERROR: VSCode extensions directory not found!" -ForegroundColor Red
    Write-Host "Make sure VSCode is installed." -ForegroundColor Red
    exit 1
}

if (Test-Path $targetDir) {
    Write-Host "Removing old installation..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $targetDir
}

Write-Host "Installing Kottg language extension..." -ForegroundColor Green
Copy-Item -Recurse -Force "vscode-kottg" $targetDir

Write-Host ""
Write-Host "SUCCESS: Extension installed!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Restart Visual Studio Code"
Write-Host "  2. Open a .kg file"
Write-Host "  3. Press Ctrl+Shift+B to compile"
Write-Host ""
