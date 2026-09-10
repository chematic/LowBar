# Installation

LowBar provides a PowerShell installer for Windows users who do not want to
clone or download the source repository.

## Install from PowerShell

Open PowerShell and run:

```powershell
iwr https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex
```

The installer:

1. queries the latest GitHub release of `chematic/LowBar`;
2. downloads only the release asset `LowBar.exe`;
3. installs it for the current Windows user under `%LOCALAPPDATA%\Programs\LowBar`;
4. creates `LowBar.lnk` in `%APPDATA%\Microsoft\Windows\Start Menu\Programs`;
5. asks whether a Desktop shortcut should also be created.

No Go source code, repository checkout, or development files are installed.

## Install a specific release

```powershell
$script = Invoke-RestMethod https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1
& ([scriptblock]::Create($script)) -Version v0.1.0
```

The release must contain an asset named `LowBar.exe`.

## Run the installer from a downloaded file

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\install.ps1
```

The installer does not require administrator privileges because it installs only
for the current user.

## Start Menu layout

After installation:

```text
%LOCALAPPDATA%\Programs\LowBar\LowBar.exe
%APPDATA%\Microsoft\Windows\Start Menu\Programs\LowBar.lnk
```

A Desktop shortcut is created only when the user answers `Y`/`Yes` (or the
French `O`/`Oui`) to the installer prompt.

## Uninstall

Delete the Start Menu shortcut and the installation directory:

```powershell
Remove-Item "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\LowBar.lnk" -Force -ErrorAction SilentlyContinue
Remove-Item "$env:LOCALAPPDATA\Programs\LowBar" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item "$([Environment]::GetFolderPath('Desktop'))\LowBar.lnk" -Force -ErrorAction SilentlyContinue
```

If LowBar is configured to start with Windows, disable **Open at boot** from
its tray menu before uninstalling.
