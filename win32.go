package main

import (
	"syscall"
	"unsafe"
)

// Minimal hand-rolled Win32 bindings. No cgo, no external modules -- just the
// handful of user32/kernel32/shell32 entry points WaveBreaker needs, kept in
// one file so the rest of the program reads like ordinary Go.

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessageW         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage          = user32.NewProc("PostQuitMessage")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procEnumChildWindows         = user32.NewProc("EnumChildWindows")
	procSendInput                = user32.NewProc("SendInput")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")

	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")

	procCheckMenuItem = user32.NewProc("CheckMenuItem")
)

const (
	wsExOverlappedWindow = 0
	wsOverlappedWindow   = 0x00CF0000
	cwUseDefault         = ^uint32(0x7FFFFFFF) // 0x80000000 as int32 sign-extended

	wmDestroy   = 0x0002
	wmClose     = 0x0010
	wmCommand   = 0x0111
	wmApp       = 0x8000
	wmNull      = 0x0000
	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205

	wmTrayIcon = wmApp + 1

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfGrayed    = 0x00000001
	mfByCommand = 0x00000000
	mfChecked   = 0x00000008
	mfUnchecked = 0x00000000

	tpmRightButton = 0x0002
	tpmLeftAlign   = 0x0000

	idiApplication = 32512 // MAKEINTRESOURCE(IDI_APPLICATION)
	idcArrow       = 32512 // MAKEINTRESOURCE(IDC_ARROW)

	idClose   = 1001
	idStartup = 1002

	processQueryLimitedInformation = 0x1000

	inputKeyboard  = 1
	keyEventFKeyUp = 0x0002
	vkReturn       = 0x0D

	hkeyCurrentUser = 0x80000001
	keyQueryValue   = 0x00000001
	keySetValue     = 0x00000002
	regSz           = 1
	errorSuccess    = 0
)

type wndClassExW struct {
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

type point struct {
	X, Y int32
}

type msg struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// NOTIFYICONDATAW, classic (pre-Vista) size -- enough for NIF_MESSAGE|NIF_ICON|NIF_TIP.
type notifyIconDataW struct {
	cbSize           uint32
	hWnd             syscall.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            syscall.Handle
	szTip            [128]uint16
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type input struct {
	typ uint32
	ki  keybdInput
	_   [8]byte // pad union out to the larger MOUSEINPUT size on amd64
}

func utf16ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

func getModuleHandle() syscall.Handle {
	h, _, _ := procGetModuleHandleW.Call(0)
	return syscall.Handle(h)
}

func loadIcon(id uintptr) syscall.Handle {
	h, _, _ := procLoadIconW.Call(0, id)
	return syscall.Handle(h)
}

// loadAppIcon loads an icon embedded as a resource in this executable
// (via rsrc_windows_amd64.syso -- see build.ps1), by resource ID.
func loadAppIcon(hInstance syscall.Handle, id uintptr) syscall.Handle {
	h, _, _ := procLoadIconW.Call(uintptr(hInstance), id)
	return syscall.Handle(h)
}

func loadCursor(id uintptr) syscall.Handle {
	h, _, _ := procLoadCursorW.Call(0, id)
	return syscall.Handle(h)
}

func registerClassExW(c *wndClassExW) uint16 {
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(c)))
	return uint16(r)
}

func createWindowExW(className, windowName string, hInstance syscall.Handle) syscall.Handle {
	h, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16ptr(className))),
		uintptr(unsafe.Pointer(utf16ptr(windowName))),
		uintptr(wsOverlappedWindow),
		0, 0, 0, 0,
		0, 0,
		uintptr(hInstance),
		0,
	)
	return syscall.Handle(h)
}

func defWindowProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func destroyWindow(hwnd syscall.Handle) {
	procDestroyWindow.Call(uintptr(hwnd))
}

func getMessage(m *msg) int32 {
	r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(m)), 0, 0, 0)
	return int32(r)
}

func translateMessage(m *msg) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(m)))
}

func dispatchMessage(m *msg) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(m)))
}

func postQuitMessage(code int32) {
	procPostQuitMessage.Call(uintptr(code))
}

func postMessage(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) {
	procPostMessageW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
}

func createPopupMenu() syscall.Handle {
	h, _, _ := procCreatePopupMenu.Call()
	return syscall.Handle(h)
}

func appendMenu(hMenu syscall.Handle, flags uint32, id uintptr, text string) {
	var textPtr uintptr
	if flags&mfSeparator == 0 {
		textPtr = uintptr(unsafe.Pointer(utf16ptr(text)))
	}
	procAppendMenuW.Call(uintptr(hMenu), uintptr(flags), id, textPtr)
}

func destroyMenu(hMenu syscall.Handle) {
	procDestroyMenu.Call(uintptr(hMenu))
}

func trackPopupMenu(hMenu syscall.Handle, x, y int32, hwnd syscall.Handle) {
	procTrackPopupMenu.Call(uintptr(hMenu), uintptr(tpmRightButton|tpmLeftAlign), uintptr(x), uintptr(y), 0, uintptr(hwnd), 0)
}

func getCursorPos() point {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

func setForegroundWindow(hwnd syscall.Handle) {
	procSetForegroundWindow.Call(uintptr(hwnd))
}

func getForegroundWindow() syscall.Handle {
	h, _, _ := procGetForegroundWindow.Call()
	return syscall.Handle(h)
}

func getWindowProcessId(hwnd syscall.Handle) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

func getWindowText(hwnd syscall.Handle) string {
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func getClassName(hwnd syscall.Handle) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

func enumChildWindows(hwnd syscall.Handle, cb uintptr, lparam uintptr) {
	procEnumChildWindows.Call(uintptr(hwnd), cb, lparam)
}

func openProcess(access uint32, pid uint32) syscall.Handle {
	h, _, _ := procOpenProcess.Call(uintptr(access), 0, uintptr(pid))
	return syscall.Handle(h)
}

func closeHandle(h syscall.Handle) {
	procCloseHandle.Call(uintptr(h))
}

func queryFullProcessImageName(h syscall.Handle) string {
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func shellNotifyIcon(dw uint32, nid *notifyIconDataW) {
	procShellNotifyIconW.Call(uintptr(dw), uintptr(unsafe.Pointer(nid)))
}

func checkMenuItem(hMenu syscall.Handle, id uint32, checked bool) {
	flag := uintptr(mfUnchecked)
	if checked {
		flag = mfChecked
	}
	procCheckMenuItem.Call(uintptr(hMenu), uintptr(id), mfByCommand|flag)
}

// regOpenRunKey opens HKCU\...\Run for the given access (keyQueryValue and/or
// keySetValue), returning 0 on failure.
func regOpenRunKey(access uint32) syscall.Handle {
	var h syscall.Handle
	path := utf16ptr(`Software\Microsoft\Windows\CurrentVersion\Run`)
	r, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(access),
		uintptr(unsafe.Pointer(&h)),
	)
	if r != errorSuccess {
		return 0
	}
	return h
}

func regCloseKey(h syscall.Handle) {
	procRegCloseKey.Call(uintptr(h))
}

func regValueExists(h syscall.Handle, name string) bool {
	namePtr := utf16ptr(name)
	var size uint32
	r, _, _ := procRegQueryValueExW.Call(uintptr(h), uintptr(unsafe.Pointer(namePtr)), 0, 0, 0, uintptr(unsafe.Pointer(&size)))
	return r == errorSuccess
}

func regSetStringValue(h syscall.Handle, name, value string) bool {
	namePtr := utf16ptr(name)
	v := syscall.StringToUTF16(value) // includes the terminating NUL
	r, _, _ := procRegSetValueExW.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(namePtr)),
		0,
		regSz,
		uintptr(unsafe.Pointer(&v[0])),
		uintptr(len(v)*2),
	)
	return r == errorSuccess
}

func regDeleteValue(h syscall.Handle, name string) {
	procRegDeleteValueW.Call(uintptr(h), uintptr(unsafe.Pointer(utf16ptr(name))))
}

func sendEnterKey() {
	var down, up input
	down.typ = inputKeyboard
	down.ki.wVk = vkReturn
	up.typ = inputKeyboard
	up.ki.wVk = vkReturn
	up.ki.dwFlags = keyEventFKeyUp

	inputs := [2]input{down, up}
	procSendInput.Call(2, uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
}
