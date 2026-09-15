#requires -Version 5.1
<#
.SYNOPSIS
    Installs or updates the official LowBar release for the current Windows user.

.DESCRIPTION
    Downloads the official LowBar release package from GitHub,
    extracts LowBar.exe, LowBarExplorerHook.dll, and the Assets directory,
    installs them under the current user's local programs directory,
    replaces an existing installation in place, restarts LowBar after an update,
    creates a Start Menu shortcut, and optionally creates a Desktop shortcut
    on first installation.

    The script never clones or downloads the source repository.
    The GitHub release asset must be named LowBar.zip and contain:

        LowBar.exe
        LowBarExplorerHook.dll
        Assets\...

.EXAMPLE
    irm https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex

.EXAMPLE
    irm https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex -Version 0.1.0
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
$assetName = 'LowBar.zip'
$apiHeaders = @{ 'User-Agent' = 'LowBar-Installer' }

$tempZip = $null
$tempExtract = $null

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
    try {
        if ([string]::IsNullOrWhiteSpace($Version)) {
            return Get-LatestRelease
        }

        $tag = $Version.Trim()

        if (-not $tag.StartsWith(
            'v',
            [System.StringComparison]::OrdinalIgnoreCase
        )) {
            $tag = "v$tag"
        }

        return Get-ReleaseByTag -Tag $tag
    }
    catch {
        $requested =
            if ([string]::IsNullOrWhiteSpace($Version)) {
                'the latest release'
            }
            else {
                "release '$Version'"
            }

        throw "Unable to find $requested on https://github.com/$repo/releases. Make sure a published GitHub Release exists."
    }
}

function Find-ReleaseAsset {
    param($Release)

    $asset = $Release.assets |
        Where-Object { $_.name -eq $assetName } |
        Select-Object -First 1

    if ($null -eq $asset) {
        throw "The release '$($Release.tag_name)' does not contain the required asset '$assetName'."
    }

    return $asset
}

function Stop-LowBarForUpdate {
    param(
        [Parameter(Mandatory = $true)]
        [string]$TargetPath
    )

    $running = @(Get-Process -Name 'LowBar' -ErrorAction SilentlyContinue)

    if ($running.Count -eq 0) {
        return
    }

    $matching = @(
        $running | Where-Object {
            try {
                $path = $_.Path
                [string]::IsNullOrWhiteSpace($path) -or
                    [string]::Equals(
                        [System.IO.Path]::GetFullPath($path),
                        [System.IO.Path]::GetFullPath($TargetPath),
                        [System.StringComparison]::OrdinalIgnoreCase
                    )
            }
            catch {
                $true
            }
        }
    )

    if ($matching.Count -eq 0) {
        return
    }

    Write-Step 'Stopping the running LowBar instance for the update...'

    if (Test-Path -LiteralPath $TargetPath -PathType Leaf) {
        try {
            Start-Process `
                -FilePath $TargetPath `
                -ArgumentList '--shutdown' `
                -WindowStyle Hidden `
                -ErrorAction Stop |
                Out-Null
        }
        catch {
            Write-Host '[LowBar] Graceful shutdown request could not be sent; continuing with process shutdown.' -ForegroundColor DarkYellow
        }
    }

    for ($i = 0; $i -lt 40; $i++) {
        Start-Sleep -Milliseconds 250

        $stillRunning = @(Get-Process -Name 'LowBar' -ErrorAction SilentlyContinue | Where-Object {
            $_.Id -in $matching.Id
        })

        if ($stillRunning.Count -eq 0) {
            return
        }
    }

    Write-Host '[LowBar] LowBar did not exit within 10 seconds; stopping it to complete the update.' -ForegroundColor DarkYellow

    foreach ($process in $matching) {
        try {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
        catch {
        }
    }

    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep -Milliseconds 250

        $stillRunning = @(Get-Process -Name 'LowBar' -ErrorAction SilentlyContinue | Where-Object {
            $_.Id -in $matching.Id
        })

        if ($stillRunning.Count -eq 0) {
            return
        }
    }

    throw 'LowBar could not be stopped, so the update was cancelled.'
}

function New-Shortcut {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ShortcutPath,

        [Parameter(Mandatory = $true)]
        [string]$TargetPath,

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
        if ($null -ne $shell) {
            [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($shell)
        }
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

    if ([string]::IsNullOrWhiteSpace($env:APPDATA)) {
        throw 'The APPDATA environment variable is not available.'
    }

    Write-Step 'Checking the official GitHub release...'

    $release = Get-Release
    $asset = Find-ReleaseAsset -Release $release

    Write-Step "Selected release $($release.tag_name)."

    $tempBase = Join-Path `
        ([System.IO.Path]::GetTempPath()) `
        ("LowBar-$([Guid]::NewGuid().ToString('N'))")

    $tempZip = "$tempBase.zip"
    $tempExtract = $tempBase

    Write-Step 'Downloading LowBar.zip...'

    Invoke-WebRequest `
        -Uri $asset.browser_download_url `
        -OutFile $tempZip `
        -Headers $apiHeaders `
        -UseBasicParsing

    if (-not (Test-Path -LiteralPath $tempZip -PathType Leaf)) {
        throw 'The download completed without producing the expected ZIP archive.'
    }

    Write-Step 'Extracting the release package...'

    New-Item `
        -ItemType Directory `
        -Force `
        -Path $tempExtract |
        Out-Null

    Expand-Archive `
        -LiteralPath $tempZip `
        -DestinationPath $tempExtract `
        -Force

    $packageExe = Join-Path $tempExtract 'LowBar.exe'
    $packageHook = Join-Path $tempExtract 'LowBarExplorerHook.dll'
    $packageAssets = Join-Path $tempExtract 'Assets'

    if (-not (Test-Path -LiteralPath $packageExe -PathType Leaf)) {
        throw "The release package is invalid: 'LowBar.exe' was not found at the archive root."
    }

    if (-not (Test-Path -LiteralPath $packageHook -PathType Leaf)) {
        throw "The release package is invalid: 'LowBarExplorerHook.dll' was not found at the archive root."
    }

    if (-not (Test-Path -LiteralPath $packageAssets -PathType Container)) {
        throw "The release package is invalid: 'Assets' was not found at the archive root."
    }

    Write-Step 'Preparing the per-user installation directory...'

    New-Item `
        -ItemType Directory `
        -Force `
        -Path $InstallRoot |
        Out-Null

    $targetExe = Join-Path $InstallRoot 'LowBar.exe'
    $targetHook = Join-Path $InstallRoot 'LowBarExplorerHook.dll'
    $targetAssets = Join-Path $InstallRoot 'Assets'
    $isUpdate = Test-Path -LiteralPath $targetExe -PathType Leaf

    if ($isUpdate) {
        Write-Step 'Existing LowBar installation detected; updating it in place.'
        Stop-LowBarForUpdate -TargetPath $targetExe
    }

    Write-Step $(if ($isUpdate) { 'Replacing LowBar.exe...' } else { 'Installing LowBar.exe...' })

    Copy-Item `
        -LiteralPath $packageExe `
        -Destination $targetExe `
        -Force

    Write-Step $(if ($isUpdate) { 'Replacing LowBarExplorerHook.dll...' } else { 'Installing LowBarExplorerHook.dll...' })

    Copy-Item `
        -LiteralPath $packageHook `
        -Destination $targetHook `
        -Force

    Write-Step 'Installing application assets...'

    if (Test-Path -LiteralPath $targetAssets) {
        Remove-Item `
            -LiteralPath $targetAssets `
            -Recurse `
            -Force
    }

    Copy-Item `
        -LiteralPath $packageAssets `
        -Destination $InstallRoot `
        -Recurse `
        -Force

    if (-not (Test-Path -LiteralPath $targetExe -PathType Leaf)) {
        throw 'LowBar.exe was not installed correctly.'
    }

    if (-not (Test-Path -LiteralPath $targetHook -PathType Leaf)) {
        throw 'LowBarExplorerHook.dll was not installed correctly.'
    }

    if (-not (Test-Path -LiteralPath $targetAssets -PathType Container)) {
        throw 'The LowBar Assets directory was not installed correctly.'
    }

    $startMenuDir = Join-Path `
        $env:APPDATA `
        'Microsoft\Windows\Start Menu\Programs'

    $startMenuShortcut = Join-Path `
        $startMenuDir `
        'LowBar.lnk'

    New-Item `
        -ItemType Directory `
        -Force `
        -Path $startMenuDir |
        Out-Null

    Write-Step 'Creating the Start Menu shortcut...'

    New-Shortcut `
        -ShortcutPath $startMenuShortcut `
        -TargetPath $targetExe `
        -WorkingDirectory $InstallRoot `
        -Description 'LowBar - Windows taskbar appearance utility' `
        -IconLocation "$targetExe,0"

    $createDesktopShortcut = $false
    $desktopShortcut = $null

    if (-not $NoDesktopPrompt -and -not $isUpdate) {
        $desktopAnswer = Read-Host `
            'Add a LowBar shortcut to the Desktop? [Y/N]'

        $createDesktopShortcut =
            $desktopAnswer -match '^(Y|y|Yes|yes|O|o|Oui|oui)$'
    }

    if ($createDesktopShortcut) {
        $desktopPath = [Environment]::GetFolderPath('Desktop')

        if ([string]::IsNullOrWhiteSpace($desktopPath)) {
            throw 'The current user Desktop directory could not be determined.'
        }

        $desktopShortcut = Join-Path `
            $desktopPath `
            'LowBar.lnk'

        Write-Step 'Creating the Desktop shortcut...'

        New-Shortcut `
            -ShortcutPath $desktopShortcut `
            -TargetPath $targetExe `
            -WorkingDirectory $InstallRoot `
            -Description 'LowBar - Windows taskbar appearance utility' `
            -IconLocation "$targetExe,0"
    }

    Write-Success `
        $(if ($isUpdate) { "LowBar $($release.tag_name) updated successfully." } else { "LowBar $($release.tag_name) installed successfully." })

    if ($isUpdate) {
        Write-Step 'Starting the updated LowBar instance...'
        Start-Process `
            -FilePath $targetExe `
            -WorkingDirectory $InstallRoot `
            -ErrorAction Stop |
            Out-Null
    }

    Write-Host `
        "Start Menu: $startMenuShortcut" `
        -ForegroundColor Gray

    Write-Host `
        "Application: $targetExe" `
        -ForegroundColor Gray

    if ($createDesktopShortcut) {
        Write-Host `
            "Desktop:     $desktopShortcut" `
            -ForegroundColor Gray
    }

    Write-Host ''

    Write-Host `
        'LowBar was installed for the current Windows user only.' `
        -ForegroundColor White

    Write-Host `
        'No source code was downloaded or installed.' `
        -ForegroundColor White
}
catch {
    Write-Host `
        "[LowBar] Installation failed: $($_.Exception.Message)" `
        -ForegroundColor Red

    Write-Host `
        '[LowBar] PowerShell session was left open so you can review the error.' `
        -ForegroundColor DarkGray
}
finally {
    if ($tempZip -and (Test-Path -LiteralPath $tempZip)) {
        Remove-Item `
            -LiteralPath $tempZip `
            -Force `
            -ErrorAction SilentlyContinue
    }

    if ($tempExtract -and (Test-Path -LiteralPath $tempExtract)) {
        Remove-Item `
            -LiteralPath $tempExtract `
            -Recurse `
            -Force `
            -ErrorAction SilentlyContinue
    }
}