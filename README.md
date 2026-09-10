# LowBar

> Lightweight Windows taskbar appearance utility — native Win32, written in Go.

LowBar changes the appearance of the Windows taskbar without replacing the
Windows shell or adding a heavy customization layer.

**Normal · Opaque · Clear · Blur · Acrylic**

LowBar is designed to stay small, local, and focused: no account, no telemetry,
and no network activity from the application itself.

## Installation

### Recommended — PowerShell

Open PowerShell and run:

```powershell
iwr https://raw.githubusercontent.com/chematic/LowBar/main/install.ps1 | iex
```

The installer downloads **only the official `LowBar.exe` release asset**. It
installs the application for the current Windows user and does not clone or
install the source repository.

It creates:

```text
%LOCALAPPDATA%\Programs\LowBar\LowBar.exe
%APPDATA%\Microsoft\Windows\Start Menu\Programs\LowBar.lnk
```

During installation, PowerShell asks whether a `LowBar` shortcut should also be
created on the user's Desktop.

See [Installation](docs/INSTALLATION.md) for the full procedure and uninstall
instructions.

## Features

- Five taskbar appearance modes: **Normal, Opaque, Clear, Blur, Acrylic**
- System tray control
- Five languages: **English, Français, Español, Deutsch, Русский**
- Optional Windows startup integration
- Automatic taskbar style reapplication after Explorer refresh/restart
- Single-instance protection
- Local configuration with validation and automatic repair
- Native Win32 implementation with no UI framework dependency
- Windows-only and user-mode

## Screenshots

Add official screenshots or a short GIF here once the visual assets are ready.
A good first set is:

```text
assets/
├── screenshot-blur.png
└── screenshot-clear.png
```

## Architecture

The runtime architecture is intentionally unchanged from the original single-file
implementation. The source is split into focused files under one `main` package,
so the compiler still produces a single executable without unnecessary runtime
layers or package boundaries.

```text
LowBar/
├── cmd/
│   └── lowbar/
│       ├── main.go             # process entry point / lifecycle
│       ├── constants.go        # Win32 constants, commands, styles, languages
│       ├── types.go             # Win32 structs and application data types
│       ├── state.go             # process-global application state
│       ├── win32.go             # DLLs and Win32 procedure bindings
│       ├── helpers.go            # small shared helpers
│       ├── logging.go            # file logging and panic reporting
│       ├── config.go             # settings.ini load/save/validation
│       ├── localization.go       # menu strings
│       ├── process.go            # DPI / AppUserModelID / hidden window setup
│       ├── icon.go               # PNG/GDI+ and tray icon loading
│       ├── taskbar.go            # taskbar discovery / composition policy
│       ├── startup.go            # HKCU Run entry
│       ├── tray.go               # Shell_NotifyIcon management
│       ├── menu.go               # context menu / command dispatch
│       ├── window.go             # hidden window procedure
│       ├── message_loop.go       # Win32 message loop
│       ├── instance.go           # single-instance mutex
│       └── cleanup.go             # shutdown cleanup
├── Assets/
├── docs/
│   ├── INSTALLATION.md
│   └── REFACTORING.md
├── install.ps1
├── LICENSE
├── TRADEMARKS.md
├── README.md
└── go.mod
```

## Build

Windows / amd64:

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

## Release artifact

The PowerShell installer expects the official GitHub release to contain an
asset named exactly:

```text
LowBar.exe
```

Example release layout:

```text
LowBar.exe
Source code (zip)
Source code (tar.gz)
```

Users of the installer receive only `LowBar.exe`; the GitHub source archive is
not downloaded by the installer.

## Configuration and logs

LowBar stores user configuration under the current user's application data and
writes diagnostic logs to its configured log directory. Invalid style/language
values are validated and repaired automatically.

## Important compatibility point

The project keeps a single `main` package on purpose. Splitting this into many
independent internal packages would require exporting state and changing call
boundaries, which is unnecessary for this utility and increases the risk of
changing behavior or adding plumbing.

## Open source license

The LowBar source code is licensed under the **Apache License 2.0**. See
[`LICENSE`](LICENSE).

The Apache-2.0 license does not grant trademark rights. The `LowBar` name, logo,
and official project branding are handled separately; see
[`TRADEMARKS.md`](TRADEMARKS.md).

This means that forks can modify and redistribute the source under Apache-2.0,
but should not present modified builds as official LowBar releases or imply
endorsement by the original project.

## Contributing

Bug reports, compatibility reports, and focused pull requests are welcome.
Please include the Windows version, LowBar version, reproduction steps, and
relevant log output when reporting a problem.

## Development notes

The structural refactor is documented in
[`docs/REFACTORING.md`](docs/REFACTORING.md).
