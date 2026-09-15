# LowBar refactoring: step by step

This refactor is structural and includes a taskbar-persistence correction. The taskbar protection is event-driven and runs at the Explorer composition boundary instead of periodically polling the taskbar.

## Step 1 — Create the module

Keep the Windows executable entry point at:

```text
cmd/lowbar/main.go
```

The rest of the source lives beside it in the same package.

## Step 2 — Move constants

Everything that was a global constant moves to `constants.go`:

- window messages
- window styles
- shell notification flags
- menu flags
- registry constants
- application identifiers
- menu command IDs
- style IDs
- language IDs

No values were changed.

## Step 3 — Move structs

All Win32 and application structs move to `types.go`:

- `point`
- `wndClassEx`
- `msg`
- `notifyIconData`
- `accentPolicy`
- `compositionAttributeData`
- `config`
- `stringsTable`
- `gdiplusStartupInput`

No field names, types, or layout were changed.

## Step 4 — Move global state

`state.go` contains the same process-wide variables the original file used.

Keeping them in the same package means all existing functions can continue to access them directly. There is no new dependency injection layer and no runtime indirection.

## Step 5 — Move Win32 bindings

`win32.go` contains the `LazyDLL` handles and `NewProc` bindings.

This isolates the low-level Windows API declarations without altering how calls are made.

## Step 6 — Split functional areas

The remaining functions are grouped by responsibility:

```text
logging.go       logging / panic reporting
config.go        settings persistence and validation
localization.go  menu language strings
process.go       DPI / AppUserModelID / hidden window setup
icon.go          tray icon loading
tray.go          Shell_NotifyIcon management
taskbar.go       taskbar discovery / accent policy
startup.go       registry startup entry
menu.go          menu construction / command dispatch
window.go        hidden window callback
message_loop.go  message pump
instance.go      single-instance mutex
cleanup.go       shutdown cleanup
```

## Step 7 — Keep the entry point small

`main.go` now only orchestrates startup, initialization, message-loop execution, and cleanup.

The runtime order is intentionally the same:

```text
LockOSThread
→ open log
→ prepare process
→ single-instance mutex
→ load config
→ detect startup state
→ register TaskbarCreated
→ load tray icon
→ create hidden window
→ install tray icon
→ apply style
→ save config
→ message loop
→ cleanup
```

## Step 8 — Explorer taskbar persistence

The taskbar reset problem is handled by `explorer_integration.go` plus `explorerhook/LowBarExplorerHook.dll`. LowBar injects the helper into the Explorer process owning the taskbar. The helper intercepts Explorer calls to `SetWindowCompositionAttribute` for the taskbar and asks LowBar whether its composition policy should be protected. This occurs only when Explorer actually makes the composition call; there is no timer-based refresh loop.

The same integration is triggered again when Windows broadcasts `TaskbarCreated`, which covers Explorer/taskbar recreation.

## Step 9 — Verify before replacing the original

Build the Windows binary:

```powershell
go build -trimpath -o LowBar.exe ./cmd/lowbar
```

Then verify on Windows:

1. LowBar starts normally.
2. The tray icon appears.
3. Right-click opens the context menu.
4. Double-click keeps the existing behavior.
5. Normal / Opaque / Clear / Blur / Acrylic render as before.
6. All five languages remain available.
7. Startup state remains synchronized with the registry.
8. Explorer restart recreates the tray icon and reapplies the selected style.
9. Exiting restores the Windows taskbar default.
10. `settings.ini` is still repaired when style/language values are invalid.
11. `LowBarExplorerHook.dll` is present beside `LowBar.exe` in release installations.
12. Opening Start, fullscreen transitions, and Explorer taskbar recreation do not require a timer-based repaint loop.

## Why this structure is intentional

This is a source organization change, not a redesign. A package-per-feature architecture would be possible, but for a small Win32 utility it would force exported APIs or additional state plumbing. Keeping `package main` avoids that complexity and minimizes the chance of changing performance or behavior.

## Installation packaging

The repository also contains `install.ps1`, which installs only the official release executable for the current Windows user. The installer places the application under `%LOCALAPPDATA%\Programs\LowBar`, creates a `LowBar.lnk` Start Menu shortcut under `%APPDATA%\Microsoft\Windows\Start Menu\Programs`, and optionally creates a Desktop shortcut. It does not install the source tree.
