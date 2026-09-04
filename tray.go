package main

import (
	"syscall"
	"unsafe"
)

const windowClassName = "WaveBreakerHiddenWindow"

var (
	hwndMain syscall.Handle
	hMenu    syscall.Handle
	nid      notifyIconDataW
)

// runTray creates the (never shown) message window and its tray icon, then
// pumps the Win32 message loop until the user picks Close. Must run on the
// OS thread that will own the window -- caller is expected to have called
// runtime.LockOSThread() first.
func runTray(tooltip string) {
	hInstance := getModuleHandle()
	icon := loadAppIcon(hInstance, 1) // resource ID 1: embedded ACS-v5.ico (see build.ps1)
	if icon == 0 {
		icon = loadIcon(idiApplication) // fallback if the .syso wasn't linked in
	}
	cursor := loadCursor(idcArrow)

	wndProcCb := syscall.NewCallback(wndProc)

	wc := wndClassExW{
		style:         0,
		lpfnWndProc:   wndProcCb,
		hInstance:     hInstance,
		hIcon:         icon,
		hCursor:       cursor,
		lpszClassName: utf16ptr(windowClassName),
		hIconSm:       icon,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	registerClassExW(&wc)

	hwndMain = createWindowExW(windowClassName, "WaveBreaker", hInstance)

	hMenu = createPopupMenu()
	appendMenu(hMenu, mfString|mfGrayed, 0, "WaveBreaker - watching for BR splash")
	appendMenu(hMenu, mfSeparator, 0, "")
	appendMenu(hMenu, mfString, idStartup, "Start with Windows")
	checkMenuItem(hMenu, idStartup, isStartupEnabled())
	appendMenu(hMenu, mfSeparator, 0, "")
	appendMenu(hMenu, mfString, idClose, "Close")

	nid = notifyIconDataW{
		hWnd:             hwndMain,
		uID:              1,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: wmTrayIcon,
		hIcon:            icon,
	}
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	copy(nid.szTip[:], syscall.StringToUTF16(tooltip))
	shellNotifyIcon(nimAdd, &nid)

	var m msg
	for getMessage(&m) > 0 {
		translateMessage(&m)
		dispatchMessage(&m)
	}
}

func wndProc(hwnd syscall.Handle, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmTrayIcon:
		switch uint32(lParam) {
		case wmLButtonUp, wmRButtonUp:
			showTrayMenu(hwnd)
		}
		return 0
	case wmCommand:
		switch uint32(wParam & 0xFFFF) {
		case idClose:
			destroyWindow(hwnd)
		case idStartup:
			newState := !isStartupEnabled()
			if setStartupEnabled(newState) {
				checkMenuItem(hMenu, idStartup, newState)
			}
		}
		return 0
	case wmDestroy:
		shellNotifyIcon(nimDelete, &nid)
		if hMenu != 0 {
			destroyMenu(hMenu)
		}
		postQuitMessage(0)
		return 0
	}
	return defWindowProc(hwnd, message, wParam, lParam)
}

func showTrayMenu(hwnd syscall.Handle) {
	checkMenuItem(hMenu, idStartup, isStartupEnabled())
	pt := getCursorPos()
	// Recommended workaround (MSDN) so the popup dismisses correctly when
	// the user clicks away from it instead of choosing an item.
	setForegroundWindow(hwnd)
	trackPopupMenu(hMenu, pt.X, pt.Y, hwnd)
	postMessage(hwnd, wmNull, 0, 0)
}
