//go:build windows

package main

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
