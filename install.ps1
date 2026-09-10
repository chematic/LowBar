#requires -Version 5.1
<#!
.SYNOPSIS
    Installs the official LowBar release for the current Windows user.

.DESCRIPTION
    Downloads only the release executable from the official GitHub repository,
    installs it under the current user's local programs directory, creates a
    Start Menu shortcut, and optionally creates a Desktop shortcut.

    The script does not clone or download the source repository.

.EXAMPLE
    irm https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex
#>

[CmdletBinding()]
param(
    [string]$Version,
    [string]$InstallRoot = "$env:LOCALAPPDATA\Programs\LowBar",
    [switch]$NoDesktopPrompt
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repo = 'chematic/LowBar'
$assetName = 'LowBar.exe'
$apiHeaders = @{ 'User-Agent' = 'LowBar-Installer' }
$tempExe = $null

function Write-Step {
    param([string]$Message)
    Write-Host "[LowBar] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[LowBar] $Message" -ForegroundColor Green
}

function Get-LatestRelease {
    $uri = "https://api.github.com/repos/$repo/releases/latest"
    return Invoke-RestMethod -Uri $uri -Headers $apiHeaders -UseBasicParsing
}

function Get-ReleaseByTag {
    param([string]$Tag)
    $uri = "https://api.github.com/repos/$repo/releases/tags/$Tag"
    return Invoke-RestMethod -Uri $uri -Headers $apiHeaders -UseBasicParsing
}

function Get-Release {
    if ([string]::IsNullOrWhiteSpace($Version)) {
        return Get-LatestRelease
    }

    $tag = $Version
    if (-not $tag.StartsWith('v')) {
        $tag = "v$tag"
    }
    return Get-ReleaseByTag -Tag $tag
}

function Find-ReleaseAsset {
    param($Release)

    $asset = $Release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
    if ($null -eq $asset) {
        throw "The release '$($Release.tag_name)' does not contain the required asset '$assetName'."
    }
    return $asset
}

function New-Shortcut {
    param(
        [Parameter(Mandatory = $true)][string]$ShortcutPath,
        [Parameter(Mandatory = $true)][string]$TargetPath,
        [string]$WorkingDirectory,
        [string]$Description,
        [string]$IconLocation
    )

    $shell = New-Object -ComObject WScript.Shell
    try {
        $shortcut = $shell.CreateShortcut($ShortcutPath)
        $shortcut.TargetPath = $TargetPath
        if (-not [string]::IsNullOrWhiteSpace($WorkingDirectory)) {
            $shortcut.WorkingDirectory = $WorkingDirectory
        }
        if (-not [string]::IsNullOrWhiteSpace($Description)) {
            $shortcut.Description = $Description
        }
        if (-not [string]::IsNullOrWhiteSpace($IconLocation)) {
            $shortcut.IconLocation = $IconLocation
        }
        $shortcut.Save()
    }
    finally {
        [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($shell)
    }
}

try {
    Write-Host ''
    Write-Host 'LowBar Installer' -ForegroundColor White
    Write-Host 'Official release installer for Windows' -ForegroundColor DarkGray
    Write-Host ''

    if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
        throw 'LowBar can only be installed on Windows.'
    }

    Write-Step 'Checking the official GitHub release...'
    $release = Get-Release
    $asset = Find-ReleaseAsset -Release $release

    Write-Step "Selected release $($release.tag_name)."
    Write-Step 'Preparing the per-user installation directory...'
    New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null

    $tempExe = Join-Path ([System.IO.Path]::GetTempPath()) ("LowBar-$([Guid]::NewGuid().ToString('N')).exe")
    $targetExe = Join-Path $InstallRoot 'LowBar.exe'

    Write-Step 'Downloading LowBar.exe...'
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempExe -Headers $apiHeaders -UseBasicParsing

    if (-not (Test-Path -LiteralPath $tempExe)) {
        throw 'The download completed without producing the expected executable.'
    }

    Write-Step 'Installing LowBar.exe...'
    Move-Item -LiteralPath $tempExe -Destination $targetExe -Force

    $startMenuDir = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs'
    $startMenuShortcut = Join-Path $startMenuDir 'LowBar.lnk'
    New-Item -ItemType Directory -Force -Path $startMenuDir | Out-Null

    Write-Step 'Creating the Start Menu shortcut...'
    New-Shortcut `
        -ShortcutPath $startMenuShortcut `
        -TargetPath $targetExe `
        -WorkingDirectory $InstallRoot `
        -Description 'LowBar - Windows taskbar appearance utility' `
        -IconLocation "$targetExe,0"

    $createDesktopShortcut = $false
    if (-not $NoDesktopPrompt) {
        $desktopAnswer = Read-Host 'Add a LowBar shortcut to the Desktop? [Y/N]'
        $createDesktopShortcut = $desktopAnswer -match '^(Y|y|Yes|yes|O|o|Oui|oui)$'
    }

    if ($createDesktopShortcut) {
        $desktopPath = [Environment]::GetFolderPath('Desktop')
        $desktopShortcut = Join-Path $desktopPath 'LowBar.lnk'
        Write-Step 'Creating the Desktop shortcut...'
        New-Shortcut `
            -ShortcutPath $desktopShortcut `
            -TargetPath $targetExe `
            -WorkingDirectory $InstallRoot `
            -Description 'LowBar - Windows taskbar appearance utility' `
            -IconLocation "$targetExe,0"
    }

    Write-Success "LowBar $($release.tag_name) installed successfully."
    Write-Host "Start Menu: $startMenuShortcut" -ForegroundColor Gray
    Write-Host "Application: $targetExe" -ForegroundColor Gray
    if ($createDesktopShortcut) {
        Write-Host "Desktop:     $desktopShortcut" -ForegroundColor Gray
    }
    Write-Host ''
    Write-Host 'LowBar was installed for the current Windows user only.' -ForegroundColor White
    Write-Host 'No source code was downloaded or installed.' -ForegroundColor White
}
catch {
    Write-Host "[LowBar] Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
finally {
    if ($tempExe -and (Test-Path -LiteralPath $tempExe)) {
        Remove-Item -LiteralPath $tempExe -Force -ErrorAction SilentlyContinue
    }
}
