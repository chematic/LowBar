#requires -Version 5.1
$ErrorActionPreference = 'Stop'

$root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$source = Join-Path $PSScriptRoot 'lowbar_explorer_hook.c'
$obj = Join-Path $PSScriptRoot 'lowbar_explorer_hook.obj'
$output = Join-Path $root 'LowBarExplorerHook.dll'

function Find-Tool([string]$name) {
    $command = Get-Command $name -ErrorAction SilentlyContinue
    if ($command -and $command.Source -and (Test-Path $command.Source)) {
        return (Resolve-Path $command.Source).Path
    }

    $candidates = @()

    if ($env:LLVMInstallDir) {
        $candidates += (Join-Path $env:LLVMInstallDir "bin\$name")
    }

    if ($env:ProgramFiles) {
        $candidates += (Join-Path $env:ProgramFiles "LLVM\bin\$name")
    }

    if (${env:ProgramW6432}) {
        $candidates += (Join-Path ${env:ProgramW6432} "LLVM\bin\$name")
    }

    if (${env:ProgramFiles(x86)}) {
        $candidates += (Join-Path ${env:ProgramFiles(x86)} "LLVM\bin\$name")
    }

    if ($env:LOCALAPPDATA) {
        $candidates += (Join-Path $env:LOCALAPPDATA "Programs\LLVM\bin\$name")
    }

    $registryPaths = @(
        'HKLM:\SOFTWARE\LLVM',
        'HKLM:\SOFTWARE\WOW6432Node\LLVM',
        'HKCU:\SOFTWARE\LLVM',
        'HKCU:\SOFTWARE\WOW6432Node\LLVM'
    )

    foreach ($registryPath in $registryPaths) {
        try {
            $installDir = (Get-ItemProperty -Path $registryPath -ErrorAction Stop).InstallDir
            if ($installDir) {
                $candidates += (Join-Path $installDir "bin\$name")
            }
        } catch {
        }
    }

    foreach ($candidate in ($candidates | Select-Object -Unique)) {
        if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }

    return $null
}

$clang = Find-Tool 'clang.exe'
$lld = Find-Tool 'lld-link.exe'

if (-not $clang) {
    throw @"
LLVM clang.exe was not found.
Expected location after the official LLVM installer:
  C:\Program Files\LLVM\bin\clang.exe

Close and reopen PowerShell after installing LLVM, or verify that clang.exe exists there.
"@
}

if (-not $lld) {
    throw @"
LLVM lld-link.exe was not found.
Expected location after the official LLVM installer:
  C:\Program Files\LLVM\bin\lld-link.exe

Close and reopen PowerShell after installing LLVM, or verify that lld-link.exe exists there.
"@
}

if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
    throw "Explorer hook source not found: $source"
}

Write-Host "[LowBar] Compiler: $clang" -ForegroundColor Cyan
Write-Host "[LowBar] Linker:   $lld" -ForegroundColor Cyan
Write-Host "[LowBar] Building $output..." -ForegroundColor Cyan

if (Test-Path -LiteralPath $obj) {
    Remove-Item -LiteralPath $obj -Force
}

if (Test-Path -LiteralPath $output) {
    Remove-Item -LiteralPath $output -Force
}

& $clang `
    -target x86_64-pc-windows-msvc `
    -O2 `
    -fno-builtin `
    -ffreestanding `
    -c $source `
    -o $obj

if ($LASTEXITCODE -ne 0) {
    throw "clang failed with exit code $LASTEXITCODE."
}

& $lld `
    /dll `
    /entry:DllMain `
    /nodefaultlib `
    /machine:x64 `
    "/out:$output" `
    $obj

if ($LASTEXITCODE -ne 0) {
    throw "lld-link failed with exit code $LASTEXITCODE."
}

Remove-Item -LiteralPath $obj -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath (Join-Path $PSScriptRoot 'lowbar_explorer_hook.exp') -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath (Join-Path $PSScriptRoot 'lowbar_explorer_hook.lib') -Force -ErrorAction SilentlyContinue

Write-Host '[LowBar] Explorer hook build completed.' -ForegroundColor Green
