package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmDestroy    = 0x0002
	wmClose      = 0x0010
	wmCommand    = 0x0111
	wmApp        = 0x8000
	wmTray       = wmApp + 1
	wmShowMenu   = wmApp + 2
	wmNull       = 0x0000
	wmRButtonUp  = 0x0205
	wmLButtonUp  = 0x0202
	wmLButtonDbl = 0x0203

	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000
	wsPopup        = 0x80000000

	csHRedraw = 0x0002
	csVRedraw = 0x0001

	iconApplication = 32512
	cursorArrow     = 32512

	nifMessage         = 0x00000001
	nifIcon            = 0x00000002
	nifTip             = 0x00000004
	nimAdd             = 0x00000000
	nimModify          = 0x00000001
	nimDelete          = 0x00000002
	nimSetVersion      = 0x00000004
	notifyIconVersion4 = 4

	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfPopup     = 0x00000010
	mfChecked   = 0x00000008

	tpmLeftButton  = 0x0000
	tpmRetCmd      = 0x0100
	tpmNoNotify    = 0x0080
	tpmNoAnimation = 0x4000
	tpmRightButton = 0x0002

	messageBoxError = 0x00000010
	messageBoxInfo  = 0x00000040

	hkeyCurrentUser      = 0x80000001
	keyRead              = 0x20019
	keyWrite             = 0x20006
	regSz                = 1
	errorSuccess         = 0
	errorFileNotFound    = 2
	errorAlreadyExists   = 183
	errorClassAlreadyReg = 1410

	appUserModelID = "Pluton.LowBar"
	appName        = "LowBar"
	configDirName  = "LowBar"
	configFileName = "settings.ini"
	logFileName    = "lowbar.log"
	mutexName      = "Local\\PlutonLowBar.SingleInstance"
	windowClass    = "PlutonLowBarHiddenWindow"
	taskbarMessage = "TaskbarCreated"

	cmdStyleNormal     = 100
	cmdStyleOpaque     = 101
	cmdStyleClear      = 102
	cmdStyleBlur       = 103
	cmdStyleAcrylic    = 104
	cmdStartup         = 110
	cmdRefresh         = 111
	cmdLanguageEnglish = 120
	cmdLanguageFrench  = 121
	cmdLanguageSpanish = 122
	cmdLanguageGerman  = 123
	cmdLanguageRussian = 124
	cmdAbout           = 130
	cmdExit            = 140
)

const (
	styleNormal = iota
	styleOpaque
	styleClear
	styleBlur
	styleAcrylic
)

const (
	langEnglish = iota
	langFrench
	langSpanish
	langGerman
	langRussian
)

type point struct{ x, y int32 }

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type notifyIconData struct {
	cbSize            uint32
	hWnd              syscall.Handle
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             syscall.Handle
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          [16]byte
	hBalloonIcon      syscall.Handle
}

type accentPolicy struct {
	state     uint32
	flags     uint32
	gradient  uint32
	animation uint32
}

type compositionAttributeData struct {
	attribute uint32
	data      uintptr
	dataSize  uintptr
}

type config struct {
	style    int
	language int
	startup  bool
}

type stringsTable struct {
	style, normal, opaque, clear, blur, acrylic string
	openBoot, refresh, language                 string
	english, french, spanish, german, russian   string
	about, exit, aboutBody, aboutTitle          string
}

type gdiplusStartupInput struct {
	gdiplusVersion           uint32
	debugEventCallback       uintptr
	suppressBackgroundThread int32
	suppressExternalCodecs   int32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdiPlus  = syscall.NewLazyDLL("gdiplus.dll")

	registerClassExW          = user32.NewProc("RegisterClassExW")
	createWindowExW           = user32.NewProc("CreateWindowExW")
	defWindowProcW            = user32.NewProc("DefWindowProcW")
	destroyWindow             = user32.NewProc("DestroyWindow")
	getMessageW               = user32.NewProc("GetMessageW")
	translateMessage          = user32.NewProc("TranslateMessage")
	dispatchMessageW          = user32.NewProc("DispatchMessageW")
	postQuitMessage           = user32.NewProc("PostQuitMessage")
	loadIconW                 = user32.NewProc("LoadIconW")
	loadCursorW               = user32.NewProc("LoadCursorW")
	setProcessDPIAware        = user32.NewProc("SetProcessDPIAware")
	setProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
	setThreadDpiAwareness     = user32.NewProc("SetThreadDpiAwarenessContext")
	setWindowComposition      = user32.NewProc("SetWindowCompositionAttribute")
	trackPopupMenuEx          = user32.NewProc("TrackPopupMenuEx")
	findWindowW               = user32.NewProc("FindWindowW")
	findWindowExW             = user32.NewProc("FindWindowExW")
	getCursorPos              = user32.NewProc("GetCursorPos")
	createPopupMenu           = user32.NewProc("CreatePopupMenu")
	appendMenuW               = user32.NewProc("AppendMenuW")
	destroyMenu               = user32.NewProc("DestroyMenu")
	checkMenuItem             = user32.NewProc("CheckMenuItem")
	setForegroundWindow       = user32.NewProc("SetForegroundWindow")
	postMessageW              = user32.NewProc("PostMessageW")
	messageBoxW               = user32.NewProc("MessageBoxW")
	registerWindowMessageW    = user32.NewProc("RegisterWindowMessageW")

	createMutexW     = kernel32.NewProc("CreateMutexW")
	closeHandle      = kernel32.NewProc("CloseHandle")
	getLastError     = kernel32.NewProc("GetLastError")
	getModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	regOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	regSetValueExW   = advapi32.NewProc("RegSetValueExW")
	regDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	regQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	regCloseKey      = advapi32.NewProc("RegCloseKey")

	shellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	setCurrentProcessAppID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")

	gdiplusStartup            = gdiPlus.NewProc("GdiplusStartup")
	gdiplusShutdown           = gdiPlus.NewProc("GdiplusShutdown")
	gdipCreateBitmapFromFile  = gdiPlus.NewProc("GdipCreateBitmapFromFile")
	gdipCreateHICONFromBitmap = gdiPlus.NewProc("GdipCreateHICONFromBitmap")
	gdipDisposeImage          = gdiPlus.NewProc("GdipDisposeImage")

	globalConfig   *config
	trayData       *notifyIconData
	trayIcon       syscall.Handle
	customTrayIcon bool
	taskbarMsg     uint32
	processMutex   syscall.Handle
	processStyle   = styleNormal
	styleApplied   bool
	logHandle      *os.File
	shuttingDown   bool
	menuPosted     bool
)

func utf16ptr(s string) *uint16        { p, _ := syscall.UTF16PtrFromString(s); return p }
func utf16from(s string, dst []uint16) { v, _ := syscall.UTF16FromString(s); copy(dst, v) }
func makeConfig() config               { return config{style: styleBlur, language: langEnglish, startup: false} }

func appDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, configDirName)
}
func configPath() string { return filepath.Join(appDataDir(), configFileName) }
func logDir() string     { return filepath.Join(os.TempDir(), configDirName) }
func logPath() string    { return filepath.Join(logDir(), logFileName) }

func openLog() {
	if err := os.MkdirAll(logDir(), 0700); err != nil {
		return
	}
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		logHandle = f
	}
}
func logEvent(level, area, message string) {
	if logHandle == nil {
		return
	}
	line := fmt.Sprintf("%s [%s] [%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05.000 -0700"), level, area, message)
	_, _ = logHandle.WriteString(line)
	_ = logHandle.Sync()
}
func logLine(s string) { logEvent("INFO", "general", s) }
func logError(area, message string, err error) {
	if err == nil {
		logEvent("ERROR", area, message)
		return
	}
	logEvent("ERROR", area, message+" | error="+err.Error())
}
func logWin32Error(area, operation string, procErr error) {
	last, _, _ := getLastError.Call()
	if last == 0 {
		logEvent("ERROR", area, fmt.Sprintf("operation=%s | procError=%v | win32LastError=0", operation, procErr))
		return
	}
	logEvent("ERROR", area, fmt.Sprintf("operation=%s | procError=%v | win32LastError=%d (%v)", operation, procErr, last, syscall.Errno(last)))
}
func executablePath() string {
	e, err := os.Executable()
	if err != nil {
		return "<unavailable: " + err.Error() + ">"
	}
	return e
}
func logContext() {
	logEvent("INFO", "startup", fmt.Sprintf("version=dev | go=%s | arch=%s | os=%s | pid=%d | exe=%s | temp=%s | config=%s | log=%s", runtime.Version(), runtime.GOARCH, runtime.GOOS, os.Getpid(), executablePath(), os.TempDir(), configPath(), logPath()))
}
func closeLog() {
	if logHandle != nil {
		_ = logHandle.Close()
		logHandle = nil
	}
}
func recoverPanic(area string) {
	if v := recover(); v != nil {
		logEvent("FATAL", area, fmt.Sprintf("panic=%v | stack=\n%s", v, debug.Stack()))
	}
}

func validStyle(v int) bool    { return v >= styleNormal && v <= styleAcrylic }
func validLanguage(v int) bool { return v >= langEnglish && v <= langRussian }

func loadConfig() config {
	cfg := makeConfig()
	file, err := os.Open(configPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logEvent("INFO", "config.load", "configuration file does not exist; defaults used")
		} else {
			logError("config.load", "unable to open configuration", err)
		}
		return cfg
	}
	defer file.Close()

	repaired := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "style":
			v, e := strconv.Atoi(value)
			if e != nil || !validStyle(v) {
				logEvent("ERROR", "config.load", fmt.Sprintf("invalid style=%q; using default=%d", value, cfg.style))
				repaired = true
			} else {
				cfg.style = v
			}
		case "language":
			v, e := strconv.Atoi(value)
			if e != nil || !validLanguage(v) {
				logEvent("ERROR", "config.load", fmt.Sprintf("invalid language=%q; using default=%d", value, cfg.language))
				repaired = true
			} else {
				cfg.language = v
			}
		case "startup":
			cfg.startup = value == "1" || strings.EqualFold(value, "true")
		}
	}
	if err := scanner.Err(); err != nil {
		logError("config.load", "scanner failed", err)
		repaired = true
	}
	if repaired {
		_ = saveConfig(cfg)
	}
	logEvent("INFO", "config.load", fmt.Sprintf("loaded style=%d language=%d startup=%t", cfg.style, cfg.language, cfg.startup))
	return cfg
}

func saveConfig(cfg config) error {
	if !validStyle(cfg.style) {
		cfg.style = styleBlur
	}
	if !validLanguage(cfg.language) {
		cfg.language = langEnglish
	}
	if err := os.MkdirAll(filepath.Dir(configPath()), 0700); err != nil {
		return err
	}
	content := fmt.Sprintf("style=%d\nlanguage=%d\nstartup=%d\n", cfg.style, cfg.language, boolInt(cfg.startup))
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, configPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	logEvent("INFO", "config.save", fmt.Sprintf("saved style=%d language=%d startup=%t path=%s", cfg.style, cfg.language, cfg.startup, configPath()))
	return nil
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func texts(lang int) stringsTable {
	if !validLanguage(lang) {
		logEvent("ERROR", "localization", fmt.Sprintf("invalid language index=%d; falling back to English", lang))
		lang = langEnglish
	}
	table := []stringsTable{
		{style: "Style", normal: "Normal", opaque: "Opaque", clear: "Clear", blur: "Blur", acrylic: "Acrylic", openBoot: "Open at boot", refresh: "Refresh taskbar", language: "Language", english: "English", french: "French", spanish: "Spanish", german: "German", russian: "Russian", about: "About", exit: "Exit", aboutBody: "LowBar\n\nA lightweight Windows taskbar appearance utility.\nNo telemetry. No network activity. User-mode Win32 application.", aboutTitle: "About LowBar"},
		{style: "Style", normal: "Normal", opaque: "Opaque", clear: "Clair", blur: "Flou", acrylic: "Acrylique", openBoot: "Lancer au démarrage", refresh: "Actualiser la barre des tâches", language: "Langue", english: "Anglais", french: "Français", spanish: "Espagnol", german: "Allemand", russian: "Russe", about: "À propos", exit: "Quitter", aboutBody: "LowBar\n\nUtilitaire léger pour l’apparence de la barre des tâches Windows.\nAucune télémétrie. Aucune activité réseau. Application Win32 en espace utilisateur.", aboutTitle: "À propos de LowBar"},
		{style: "Estilo", normal: "Normal", opaque: "Opaco", clear: "Transparente", blur: "Desenfoque", acrylic: "Acrílico", openBoot: "Abrir al iniciar", refresh: "Actualizar barra de tareas", language: "Idioma", english: "Inglés", french: "Francés", spanish: "Español", german: "Alemán", russian: "Ruso", about: "Acerca de", exit: "Salir", aboutBody: "LowBar\n\nUtilidad ligera para la apariencia de la barra de tareas de Windows.\nSin telemetría. Sin actividad de red. Aplicación Win32 en modo usuario.", aboutTitle: "Acerca de LowBar"},
		{style: "Stil", normal: "Normal", opaque: "Deckend", clear: "Klar", blur: "Unschärfe", acrylic: "Acryl", openBoot: "Beim Start öffnen", refresh: "Taskleiste aktualisieren", language: "Sprache", english: "Englisch", french: "Französisch", spanish: "Spanisch", german: "Deutsch", russian: "Russisch", about: "Über", exit: "Beenden", aboutBody: "LowBar\n\nLeichtes Dienstprogramm für das Erscheinungsbild der Windows-Taskleiste.\nKeine Telemetrie. Keine Netzwerkaktivität. Win32-Anwendung im Benutzermodus.", aboutTitle: "Über LowBar"},
		{style: "Стиль", normal: "Обычный", opaque: "Непрозрачный", clear: "Прозрачный", blur: "Размытие", acrylic: "Акрил", openBoot: "Запускать при входе", refresh: "Обновить панель задач", language: "Язык", english: "Английский", french: "Французский", spanish: "Испанский", german: "Немецкий", russian: "Русский", about: "О программе", exit: "Выход", aboutBody: "LowBar\n\nЛёгкая утилита для оформления панели задач Windows.\nБез телеметрии. Без сетевой активности. Win32-приложение в пользовательском режиме.", aboutTitle: "О программе LowBar"},
	}
	return table[lang]
}

func showError(title string, err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	logError("ui.error", title, err)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(utf16ptr(err.Error()))), uintptr(unsafe.Pointer(utf16ptr(title))), messageBoxError)
}
func showInfo(owner syscall.Handle, title, text string) {
	messageBoxW.Call(uintptr(owner), uintptr(unsafe.Pointer(utf16ptr(text))), uintptr(unsafe.Pointer(utf16ptr(title))), messageBoxInfo)
}
func getModuleHandle() syscall.Handle { v, _, _ := getModuleHandleW.Call(0); return syscall.Handle(v) }

func prepareProcess() {
	if ret, _, err := setCurrentProcessAppID.Call(uintptr(unsafe.Pointer(utf16ptr(appUserModelID)))); ret == 0 {
		logWin32Error("process", "SetCurrentProcessExplicitAppUserModelID", err)
	}
	if ret, _, err := setProcessDpiAwarenessCtx.Call(uintptr(^uintptr(0))); ret == 0 {
		logWin32Error("process", "SetProcessDpiAwarenessContext", err)
	}
	if ret, _, err := setProcessDPIAware.Call(); ret == 0 {
		logWin32Error("process", "SetProcessDPIAware", err)
	}
	if ret, _, err := setThreadDpiAwareness.Call(uintptr(^uintptr(0))); ret == 0 {
		logWin32Error("process", "SetThreadDpiAwarenessContext", err)
	}
}

func createHiddenWindow() (syscall.Handle, error) {
	instance := getModuleHandle()
	className := utf16ptr(windowClass)
	cursor, _, _ := loadCursorW.Call(0, cursorArrow)
	wc := wndClassEx{cbSize: uint32(unsafe.Sizeof(wndClassEx{})), style: csHRedraw | csVRedraw, lpfnWndProc: syscall.NewCallback(windowProc), hInstance: instance, hIcon: trayIcon, hCursor: syscall.Handle(cursor), lpszClassName: className, hIconSm: trayIcon}
	ret, _, callErr := registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 && callErr != nil && !errors.Is(callErr, syscall.Errno(errorClassAlreadyReg)) {
		return 0, callErr
	}
	hwnd, _, err := createWindowExW.Call(wsExToolWindow|wsExNoActivate, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16ptr(appName))), wsPopup, 0, 0, 0, 0, 0, 0, uintptr(instance), 0)
	if hwnd == 0 {
		if err == nil {
			err = syscall.EINVAL
		}
		return 0, err
	}
	return syscall.Handle(hwnd), nil
}

func iconPath() string {
	e, err := os.Executable()
	if err != nil {
		return filepath.Join("Assets", "icon.png")
	}
	return filepath.Join(filepath.Dir(e), "Assets", "icon.png")
}

func loadPNGIcon(path string) (syscall.Handle, error) {
	var token uintptr
	input := gdiplusStartupInput{gdiplusVersion: 1}
	status, _, callErr := gdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&input)), 0)
	if status != 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("GDI+ startup failed: status %d", status)
	}
	defer gdiplusShutdown.Call(token)
	var bitmap uintptr
	utfPath := utf16ptr(path)
	status, _, callErr = gdipCreateBitmapFromFile.Call(uintptr(unsafe.Pointer(utfPath)), uintptr(unsafe.Pointer(&bitmap)))
	if status != 0 || bitmap == 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("could not load icon: %s", path)
	}
	defer gdipDisposeImage.Call(bitmap)
	var hicon uintptr
	status, _, callErr = gdipCreateHICONFromBitmap.Call(bitmap, uintptr(unsafe.Pointer(&hicon)))
	if status != 0 || hicon == 0 {
		if callErr != nil {
			return 0, callErr
		}
		return 0, fmt.Errorf("could not create icon from PNG: %s", path)
	}
	return syscall.Handle(hicon), nil
}

func loadTrayIcon() error {
	path := iconPath()
	icon, err := loadPNGIcon(path)
	if err == nil {
		trayIcon = icon
		customTrayIcon = true
		return nil
	}
	logLine("custom icon unavailable: " + err.Error())
	fallback, _, fallbackErr := loadIconW.Call(0, iconApplication)
	if fallback == 0 {
		if fallbackErr != nil {
			return fallbackErr
		}
		return err
	}
	trayIcon = syscall.Handle(fallback)
	customTrayIcon = false
	return nil
}
func destroyTrayIconHandle() {
	if trayIcon == 0 || !customTrayIcon {
		trayIcon = 0
		customTrayIcon = false
		return
	}
	user32.NewProc("DestroyIcon").Call(uintptr(trayIcon))
	trayIcon = 0
	customTrayIcon = false
}

func installTrayIcon(hwnd syscall.Handle) (notifyIconData, error) {
	var data notifyIconData
	data.cbSize = uint32(unsafe.Sizeof(data))
	data.hWnd = hwnd
	data.uID = 1
	data.uFlags = nifMessage | nifIcon | nifTip
	data.uCallbackMessage = wmTray
	data.hIcon = trayIcon
	utf16from(appName, data.szTip[:])
	ret, _, err := shellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	if ret == 0 {
		logWin32Error("tray.install", "Shell_NotifyIconW(NIM_ADD)", err)
		if err != nil {
			return data, err
		}
		return data, errors.New("Shell_NotifyIconW failed")
	}
	data.uTimeoutOrVersion = notifyIconVersion4
	if ret, _, err = shellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&data))); ret == 0 {
		logWin32Error("tray.install", "Shell_NotifyIconW(NIM_SETVERSION)", err)
	}
	return data, nil
}
func removeTrayIcon(data *notifyIconData) {
	if data == nil || data.hWnd == 0 {
		return
	}
	shellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(data)))
}

func reinstallTrayIcon(hwnd syscall.Handle) {
	if trayData != nil {
		removeTrayIcon(trayData)
	}
	data, err := installTrayIcon(hwnd)
	if err != nil {
		logError("tray.reinstall", "unable to reinstall tray icon", err)
		trayData = nil
		return
	}
	trayData = &data
	logEvent("INFO", "tray.reinstall", "tray icon reinstalled after Explorer refresh")
}

func taskbarWindows() []syscall.Handle {
	var result []syscall.Handle
	primaryName := utf16ptr("Shell_TrayWnd")
	secondaryName := utf16ptr("Shell_SecondaryTrayWnd")
	primary, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(primaryName)), 0)
	if primary != 0 {
		result = append(result, syscall.Handle(primary))
	}
	var after uintptr
	for {
		hwnd, _, _ := findWindowExW.Call(0, after, uintptr(unsafe.Pointer(secondaryName)), 0)
		if hwnd == 0 {
			break
		}
		result = append(result, syscall.Handle(hwnd))
		after = hwnd
	}
	return result
}

func accentForStyle(style int) accentPolicy {
	switch style {
	case styleNormal:
		return accentPolicy{state: 0, flags: 0, gradient: 0, animation: 0}
	case styleOpaque:
		return accentPolicy{state: 1, flags: 2, gradient: 0xFF111111}
	case styleClear:
		return accentPolicy{state: 2, flags: 2, gradient: 0x00000000}
	case styleBlur:
		return accentPolicy{state: 3, flags: 2, gradient: 0x55000000}
	case styleAcrylic:
		return accentPolicy{state: 4, flags: 2, gradient: 0xAA000000}
	default:
		return accentPolicy{state: 0}
	}
}

func setAccent(hwnd syscall.Handle, style int) bool {
	if hwnd == 0 {
		return false
	}
	policy := accentForStyle(style)
	data := compositionAttributeData{attribute: 19, data: uintptr(unsafe.Pointer(&policy)), dataSize: unsafe.Sizeof(policy)}
	ret, _, err := setWindowComposition.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&data)))
	if ret == 0 {
		logWin32Error("taskbar.setAccent", fmt.Sprintf("SetWindowCompositionAttribute hwnd=%d style=%d", hwnd, style), err)
		return false
	}
	return true
}

func applyStyle(style int) {
	defer recoverPanic("applyStyle")
	if !validStyle(style) {
		logEvent("ERROR", "taskbar.applyStyle", fmt.Sprintf("invalid style=%d; falling back to Normal", style))
		style = styleNormal
	}
	windows := taskbarWindows()
	if len(windows) == 0 {
		logEvent("ERROR", "taskbar.applyStyle", "no taskbar windows found")
		return
	}
	success := 0
	for _, hwnd := range windows {
		if setAccent(hwnd, style) {
			success++
		}
	}
	if success == len(windows) {
		processStyle = style
		styleApplied = style != styleNormal
		logEvent("INFO", "taskbar.applyStyle", fmt.Sprintf("style=%d applied to %d taskbar window(s)", style, success))
	} else {
		logEvent("ERROR", "taskbar.applyStyle", fmt.Sprintf("style=%d partially applied success=%d total=%d", style, success, len(windows)))
	}
}

func restoreTaskbar() {
	defer recoverPanic("restoreTaskbar")
	if !styleApplied {
		return
	}
	windows := taskbarWindows()
	for _, hwnd := range windows {
		_ = setAccent(hwnd, styleNormal)
	}
	styleApplied = false
	processStyle = styleNormal
	logEvent("INFO", "taskbar.restore", "taskbar restored to Normal/Windows default")
}

func setStartup(enabled bool) error {
	keyPath := utf16ptr("Software\\Microsoft\\Windows\\CurrentVersion\\Run")
	valueName := utf16ptr(appName)
	var key syscall.Handle
	ret, _, _ := regOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, keyWrite, uintptr(unsafe.Pointer(&key)))
	if ret != errorSuccess {
		return syscall.Errno(ret)
	}
	defer regCloseKey.Call(uintptr(key))
	if !enabled {
		ret, _, _ = regDeleteValueW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)))
		if ret == errorFileNotFound {
			return nil
		}
		if ret != errorSuccess {
			return syscall.Errno(ret)
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	value, err := syscall.UTF16FromString("\"" + exe + "\"")
	if err != nil {
		return err
	}
	valueBytes := unsafe.Slice((*byte)(unsafe.Pointer(&value[0])), len(value)*2)
	ret, _, _ = regSetValueExW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)), 0, regSz, uintptr(unsafe.Pointer(&valueBytes[0])), uintptr(len(valueBytes)))
	if ret != errorSuccess {
		return syscall.Errno(ret)
	}
	return nil
}
func isStartupEnabled() bool {
	keyPath := utf16ptr("Software\\Microsoft\\Windows\\CurrentVersion\\Run")
	valueName := utf16ptr(appName)
	var key syscall.Handle
	ret, _, _ := regOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(keyPath)), 0, keyRead, uintptr(unsafe.Pointer(&key)))
	if ret != errorSuccess {
		return false
	}
	defer regCloseKey.Call(uintptr(key))
	var valueType uint32
	var data [2048]byte
	size := uint32(len(data))
	ret, _, _ = regQueryValueExW.Call(uintptr(key), uintptr(unsafe.Pointer(valueName)), 0, uintptr(unsafe.Pointer(&valueType)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)))
	return ret == errorSuccess && size > 0
}

func appendMenu(menu, flags, id uintptr, text string) bool {
	ret, _, err := appendMenuW.Call(menu, flags, id, uintptr(unsafe.Pointer(utf16ptr(text))))
	if ret == 0 {
		logWin32Error("tray.menu", fmt.Sprintf("AppendMenuW menu=%d id=%d text=%q", menu, id, text), err)
		return false
	}
	return true
}
func menuCheckFlag(active bool) uintptr {
	if active {
		return mfChecked
	}
	return 0
}

func showTrayMenu(hwnd syscall.Handle, cfg *config) {
	defer recoverPanic("tray.menu")
	if cfg == nil {
		return
	}
	if !validStyle(cfg.style) {
		cfg.style = styleBlur
	}
	if !validLanguage(cfg.language) {
		cfg.language = langEnglish
	}
	labels := texts(cfg.language)
	var menu, styleMenu, langMenu uintptr
	menu, _, _ = createPopupMenu.Call()
	styleMenu, _, _ = createPopupMenu.Call()
	langMenu, _, _ = createPopupMenu.Call()
	if menu == 0 || styleMenu == 0 || langMenu == 0 {
		if menu != 0 {
			destroyMenu.Call(menu)
		}
		if styleMenu != 0 {
			destroyMenu.Call(styleMenu)
		}
		if langMenu != 0 {
			destroyMenu.Call(langMenu)
		}
		return
	}
	defer destroyMenu.Call(menu)
	defer destroyMenu.Call(styleMenu)
	defer destroyMenu.Call(langMenu)

	items := []struct {
		m  uintptr
		f  uintptr
		id uintptr
		t  string
	}{
		{styleMenu, mfString, cmdStyleNormal, labels.normal}, {styleMenu, mfString, cmdStyleOpaque, labels.opaque}, {styleMenu, mfString, cmdStyleClear, labels.clear}, {styleMenu, mfString, cmdStyleBlur, labels.blur}, {styleMenu, mfString, cmdStyleAcrylic, labels.acrylic},
		{langMenu, mfString, cmdLanguageEnglish, labels.english}, {langMenu, mfString, cmdLanguageFrench, labels.french}, {langMenu, mfString, cmdLanguageSpanish, labels.spanish}, {langMenu, mfString, cmdLanguageGerman, labels.german}, {langMenu, mfString, cmdLanguageRussian, labels.russian},
	}
	for _, it := range items {
		if !appendMenu(it.m, it.f, it.id, it.t) {
			return
		}
	}
	if !appendMenu(menu, mfPopup, styleMenu, labels.style) {
		return
	}
	if !appendMenu(menu, mfString, cmdStartup, labels.openBoot) {
		return
	}
	if !appendMenu(menu, mfString, cmdRefresh, labels.refresh) {
		return
	}
	if !appendMenu(menu, mfPopup, langMenu, labels.language) {
		return
	}
	if !appendMenu(menu, mfSeparator, 0, "") {
		return
	}
	if !appendMenu(menu, mfString, cmdAbout, labels.about) {
		return
	}
	if !appendMenu(menu, mfString, cmdExit, labels.exit) {
		return
	}

	checkMenuItem.Call(styleMenu, cmdStyleNormal, menuCheckFlag(cfg.style == styleNormal))
	checkMenuItem.Call(styleMenu, cmdStyleOpaque, menuCheckFlag(cfg.style == styleOpaque))
	checkMenuItem.Call(styleMenu, cmdStyleClear, menuCheckFlag(cfg.style == styleClear))
	checkMenuItem.Call(styleMenu, cmdStyleBlur, menuCheckFlag(cfg.style == styleBlur))
	checkMenuItem.Call(styleMenu, cmdStyleAcrylic, menuCheckFlag(cfg.style == styleAcrylic))
	checkMenuItem.Call(langMenu, cmdLanguageEnglish, menuCheckFlag(cfg.language == langEnglish))
	checkMenuItem.Call(langMenu, cmdLanguageFrench, menuCheckFlag(cfg.language == langFrench))
	checkMenuItem.Call(langMenu, cmdLanguageSpanish, menuCheckFlag(cfg.language == langSpanish))
	checkMenuItem.Call(langMenu, cmdLanguageGerman, menuCheckFlag(cfg.language == langGerman))
	checkMenuItem.Call(langMenu, cmdLanguageRussian, menuCheckFlag(cfg.language == langRussian))
	checkMenuItem.Call(menu, cmdStartup, menuCheckFlag(cfg.startup))

	var cursor point
	if ret, _, err := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ret == 0 {
		logWin32Error("tray.menu", "GetCursorPos", err)
		return
	}
	if ret, _, err := setForegroundWindow.Call(uintptr(hwnd)); ret == 0 {
		logWin32Error("tray.menu", "SetForegroundWindow", err)
		return
	}
	selected, _, err := trackPopupMenuEx.Call(menu, tpmRetCmd|tpmNoNotify|tpmNoAnimation|tpmLeftButton|tpmRightButton, uintptr(cursor.x), uintptr(cursor.y), uintptr(hwnd), 0)
	// Required by TrackPopupMenuEx so the shell/menu focus is released cleanly.
	postMessageW.Call(uintptr(hwnd), wmNull, 0, 0)
	if selected != 0 {
		selectCommand(hwnd, uint32(selected), cfg)
	} else if err != nil {
		logWin32Error("tray.menu", "TrackPopupMenuEx", err)
	}
}

func selectCommand(hwnd syscall.Handle, command uint32, cfg *config) {
	defer recoverPanic("command")
	switch command {
	case cmdStyleNormal, cmdStyleOpaque, cmdStyleClear, cmdStyleBlur, cmdStyleAcrylic:
		cfg.style = int(command - cmdStyleNormal)
		applyStyle(cfg.style)
		if err := saveConfig(*cfg); err != nil {
			logError("config.save", "unable to save style", err)
		}
	case cmdStartup:
		next := !cfg.startup
		if err := setStartup(next); err != nil {
			showError(appName, err)
			return
		}
		cfg.startup = next
		if err := saveConfig(*cfg); err != nil {
			logError("config.save", "unable to save startup state", err)
		}
	case cmdRefresh:
		applyStyle(cfg.style)
	case cmdLanguageEnglish, cmdLanguageFrench, cmdLanguageSpanish, cmdLanguageGerman, cmdLanguageRussian:
		cfg.language = int(command - cmdLanguageEnglish)
		if err := saveConfig(*cfg); err != nil {
			logError("config.save", "unable to save language", err)
		}
	case cmdAbout:
		labels := texts(cfg.language)
		showInfo(hwnd, labels.aboutTitle, labels.aboutBody)
	case cmdExit:
		requestExit(hwnd)
	}
}

func requestExit(hwnd syscall.Handle) {
	if shuttingDown {
		return
	}
	shuttingDown = true
	postMessageW.Call(uintptr(hwnd), wmClose, 0, 0)
}

func windowProc(hwnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	defer recoverPanic("window procedure")
	switch message {
	case wmTray:
		event := uint32(lParam & 0xFFFF)
		if event == wmRButtonUp || event == wmLButtonDbl {
			if globalConfig != nil && !shuttingDown && !menuPosted {
				menuPosted = true
				postMessageW.Call(hwnd, wmShowMenu, 0, 0)
			}
			return 0
		}
		return 0
	case wmShowMenu:
		menuPosted = false
		if globalConfig != nil && !shuttingDown {
			showTrayMenu(syscall.Handle(hwnd), globalConfig)
		}
		return 0
	case wmCommand:
		return 0
	case wmClose:
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		postQuitMessage.Call(0)
		return 0
	}
	if taskbarMsg != 0 && message == taskbarMsg && globalConfig != nil && !shuttingDown {
		reinstallTrayIcon(syscall.Handle(hwnd))
		applyStyle(globalConfig.style)
		return 0
	}
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func messageLoop() int {
	var message msg
	for {
		result, _, err := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			logWin32Error("message.loop", "GetMessageW", err)
			return 1
		}
		if result == 0 {
			return 0
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func ensureSingleInstance() syscall.Handle {
	mutex, _, _ := createMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16ptr(mutexName))))
	if mutex == 0 {
		return 0
	}
	last, _, _ := getLastError.Call()
	if last == errorAlreadyExists {
		closeHandle.Call(mutex)
		return 0
	}
	return syscall.Handle(mutex)
}

func cleanup() {
	defer recoverPanic("cleanup")
	restoreTaskbar()
	if processMutex != 0 {
		closeHandle.Call(uintptr(processMutex))
		processMutex = 0
	}
	if trayData != nil {
		removeTrayIcon(trayData)
		trayData = nil
	}
	destroyTrayIconHandle()
	closeLog()
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	openLog()
	logContext()
	defer func() {
		if v := recover(); v != nil {
			logEvent("FATAL", "main", fmt.Sprintf("panic=%v | stack=\n%s", v, debug.Stack()))
		}
		cleanup()
	}()
	prepareProcess()
	processMutex = ensureSingleInstance()
	if processMutex == 0 {
		return
	}
	cfg := loadConfig()
	startup := isStartupEnabled()
	if cfg.startup != startup {
		cfg.startup = startup
		_ = saveConfig(cfg)
	}
	globalConfig = &cfg
	taskbarRaw, _, _ := registerWindowMessageW.Call(uintptr(unsafe.Pointer(utf16ptr(taskbarMessage))))
	taskbarMsg = uint32(taskbarRaw)
	if err := loadTrayIcon(); err != nil {
		showError(appName, err)
		return
	}
	hwnd, err := createHiddenWindow()
	if err != nil {
		showError(appName, err)
		return
	}
	defer destroyWindow.Call(uintptr(hwnd))
	tray, err := installTrayIcon(hwnd)
	if err != nil {
		showError(appName, err)
		return
	}
	trayData = &tray
	applyStyle(cfg.style)
	if err := saveConfig(cfg); err != nil {
		logError("config.save", "initial config save failed", err)
	}
	_ = messageLoop()
}
