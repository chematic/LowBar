//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

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
