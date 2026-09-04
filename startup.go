package main

import "os"

// "Start with Windows" is implemented as a per-user HKCU Run key entry --
// no admin rights needed, and it only affects the current Windows user.
const startupValueName = "WaveBreaker"

func isStartupEnabled() bool {
	h := regOpenRunKey(keyQueryValue)
	if h == 0 {
		return false
	}
	defer regCloseKey(h)
	return regValueExists(h, startupValueName)
}

// setStartupEnabled adds or removes the Run key entry, pointing it at this
// running executable's current path (quoted, in case it ever contains a
// space).
func setStartupEnabled(enable bool) bool {
	h := regOpenRunKey(keySetValue)
	if h == 0 {
		return false
	}
	defer regCloseKey(h)

	if !enable {
		regDeleteValue(h, startupValueName)
		return true
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	return regSetStringValue(h, startupValueName, `"`+exePath+`"`)
}
