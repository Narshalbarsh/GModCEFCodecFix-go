package ui

import (
	"os"
	"syscall"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole     = kernel32.NewProc("AttachConsole")
	procGetConsoleWindow  = kernel32.NewProc("GetConsoleWindow")
	procGetStdHandle      = kernel32.NewProc("GetStdHandle")
	ATTACH_PARENT_PROCESS = ^uint32(0) // Special value to attach to the parent process
)

const (
	STD_OUTPUT_HANDLE = -11 & 0xFFFFFFFF // Corresponds to (DWORD)-11
	STD_ERROR_HANDLE  = -12 & 0xFFFFFFFF // Corresponds to (DWORD)-12
)

// Try to get output in the terminal if we launched it from there,
// but don't open a new console if we double click the exe file
func AttachToConsole() {
	consoleHandle, _, _ := procGetConsoleWindow.Call()
	if consoleHandle == 0 {
		attached, _, _ := procAttachConsole.Call(uintptr(ATTACH_PARENT_PROCESS))
		if attached == 0 {
			return
		}
		stdOutHandle, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))
		stdErrHandle, _, _ := procGetStdHandle.Call(uintptr(STD_ERROR_HANDLE))
		if stdOutHandle != uintptr(syscall.InvalidHandle) {
			os.Stdout = os.NewFile(stdOutHandle, "stdout")
		}
		if stdErrHandle != uintptr(syscall.InvalidHandle) {
			os.Stderr = os.NewFile(stdErrHandle, "stderr")
		}
	}
}
