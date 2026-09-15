# LowBar

> Lightweight Windows taskbar appearance utility, native Win32, written in Go.

LowBar changes the appearance of the Windows taskbar without replacing the Windows shell or adding a heavy customization layer.

**Normal · Opaque · Clear · Blur · Acrylic**

LowBar is designed to stay small, local, and focused: no account, no telemetry, and no network activity from the application itself.

## Installation

### Recommended: PowerShell

Open PowerShell and run:

```powershell
iwr https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex
```

The installer downloads **only the official `LowBar.zip` release asset**. It installs the application for the current Windows user and does not clone or install the source repository.

Re-running the installer updates an existing installation in place, restarts LowBar automatically, and keeps the existing user configuration.

It creates:

```text
%LOCALAPPDATA%\Programs\LowBar\LowBar.exe
%APPDATA%\Microsoft\Windows\Start Menu\Programs\LowBar.lnk
```

During installation, PowerShell asks whether a `LowBar` shortcut should also be created on the user's Desktop.

See [`docs/INSTALLATION.md`](docs/INSTALLATION.md) for the full procedure and uninstall instructions.

## Features

- Five taskbar appearance modes: **Normal, Opaque, Clear, Blur, Acrylic**
- System tray control
- Five languages: **English, Français, Español, Deutsch, Русский**
- Optional Windows startup integration
- Explorer-side composition protection: LowBar intercepts Explorer's taskbar accent writes while LowBar is active
- Event-driven Explorer integration for taskbar creation/restart; no periodic taskbar timer or polling loop
- Single-instance protection
- Local configuration with validation and automatic repair
- Native Win32 implementation with no UI framework dependency
- Windows-only and user-mode

## Screenshots

Add official screenshots or a short GIF here once the visual assets are ready.

A good first set is:

```text
Assets/
├── screenshot-blur.png
└── screenshot-clear.png
```

## Architecture

The runtime stays native and focused, but the taskbar persistence path is implemented at the Explorer boundary rather than with periodic polling. The source is split into focused files under one `main` package, so the compiler still produces a single executable without unnecessary runtime layers or package boundaries.

```text
LowBar/
├── cmd/
│   ├── lowbar/
│   │   ├── main.go
│   │   ├── constants.go
│   │   ├── types.go
│   │   ├── state.go
│   │   ├── win32.go
│   │   ├── helpers.go
│   │   ├── logging.go
│   │   ├── config.go
│   │   ├── localization.go
│   │   ├── process.go
│   │   ├── icon.go
│   │   ├── taskbar.go
│   │   ├── startup.go
│   │   ├── tray.go
│   │   ├── menu.go
│   │   ├── window.go
│   │   ├── message_loop.go
│   │   ├── instance.go
│   │   ├── explorer_integration.go
│   │   └── cleanup.go
│   └── explorerhook/
│       ├── lowbar_explorer_hook.c
│       └── build.ps1
├── Assets/
├── docs/
├── scripts/
│   └── package-release.ps1
├── install.ps1
├── LICENSE
├── TRADEMARKS.md
├── README.md
└── go.mod
```

## Build

LowBar itself is written in Go and only requires the Go toolchain for development builds.

### Windows / amd64

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"

go build -trimpath -ldflags="-s -w -H=windowsgui" -o LowBar.exe ./cmd/lowbar
```

For a smaller release binary:

```powershell
go build -trimpath -ldflags="-s -w -H=windowsgui" -o LowBar.exe ./cmd/lowbar
```

The application remains Windows-only because it directly uses Win32 APIs.

### Build the Explorer hook

The Explorer hook is a native x64 Windows DLL loaded into the 64-bit Explorer process.

The hook is intentionally kept under `cmd/explorerhook` because it is a build-time component of LowBar, not a separate runtime project.

The hook source is compiled with **LLVM Clang + LLD**. The build does **not** use MSVC `cl.exe`.

#### Install LLVM

Install LLVM for Windows with WinGet:

```powershell
winget install -e --id LLVM.LLVM
```

The build script searches common LLVM installation paths automatically, including:

```text
C:\Program Files\LLVM\bin\
C:\Program Files (x86)\LLVM\bin\
%LOCALAPPDATA%\Programs\LLVM\bin\
```

A new PowerShell session may be required if LLVM was added to `PATH` by the installer.

#### Build the hook

From the repository root:

```powershell
.\cmd\explorerhook\build.ps1
```

The script uses:

```text
clang.exe
lld-link.exe
```

and produces:

```text
LowBarExplorerHook.dll
```

at the repository root.

The source build does not require Visual Studio Developer PowerShell or `cl.exe`.

Do not commit generated build artifacts such as:

```text
LowBarExplorerHook.dll
*.obj
*.exp
*.lib
```

to the source repository.

## Configuration and logs

LowBar stores user configuration under the current user's application data and writes diagnostic logs to its configured log directory.

Invalid style and language values are validated and repaired automatically.

## Taskbar persistence architecture

LowBar does not use a timer to repeatedly repaint or reset the taskbar.

While LowBar is running, the x64 `LowBarExplorerHook.dll` is loaded into the Explorer process that owns the taskbar. It detours `user32!SetWindowCompositionAttribute` and, for `WCA_ACCENT_POLICY` writes targeting `Shell_TrayWnd` or `Shell_SecondaryTrayWnd`, asks the LowBar hidden window whether LowBar is active.

Active LowBar writes are therefore protected at the point where Explorer attempts to replace them.

The protection is event-driven: taskbar creation or recreation through `TaskbarCreated` triggers the Explorer integration again. No periodic taskbar polling loop is used.

DWM remains the final compositor. This design controls Explorer's taskbar composition writes without modifying the Windows kernel.

On Windows 11 22H2 and later, important taskbar visuals moved to a XAML-based implementation. LowBar's classic `WCA_ACCENT_POLICY` protection covers the classic Win32 composition path. A fully equivalent XAML taskbar implementation requires an Explorer XAML/TAP layer similar to the approach used by TranslucentTB.

## Compatibility

LowBar targets Windows systems using the native Win32 taskbar architecture.

The primary supported targets are:

- Windows 10 x64
- Windows 11 x64

The Explorer hook is built specifically as a native x64 DLL because modern 64-bit Windows Explorer requires the injected module to match the Explorer process architecture.

## Important development point

The project keeps a single `main` package for the application on purpose.

The Explorer hook is a separate native build-time component because it must execute inside the Explorer process and therefore cannot be implemented as ordinary Go application code.

The runtime remains a single Windows executable plus the Explorer hook DLL:

```text
LowBar.exe
LowBarExplorerHook.dll
```

## Open source license

The LowBar source code is licensed under the **Apache License 2.0**. See [`LICENSE`](LICENSE).

The Apache-2.0 license does not grant trademark rights. The `LowBar` name, logo, and official project branding are handled separately; see [`TRADEMARKS.md`](TRADEMARKS.md).

This means that forks can modify and redistribute the source under Apache-2.0, but should not present modified builds as official LowBar releases or imply endorsement by the original project.

## Contributing

Bug reports, compatibility reports, and focused pull requests are welcome.

Please include the Windows version, LowBar version, reproduction steps, and relevant log output when reporting a problem.

## Development notes

The structural refactor is documented in [`docs/REFACTORING.md`](docs/REFACTORING.md).

### Explorer hook development build

The Explorer hook is a native x64 Windows DLL.

The source build uses:

```text
LLVM clang.exe
LLVM lld-link.exe
```

MSVC `cl.exe` is **not required**.

Install LLVM for Windows:

```powershell
winget install -e --id LLVM.LLVM
```

Build the hook:

```powershell
.\cmd\explorerhook\build.ps1
```

The generated `LowBarExplorerHook.dll` is placed beside `LowBar.exe` for local testing and release packaging.

End users do **not** need:

- Go
- LLVM
- Visual Studio
- MSVC
- Windows SDK

when using the official release ZIP.