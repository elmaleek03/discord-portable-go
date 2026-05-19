//go:build windows

package common

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole     = kernel32.NewProc("AllocConsole")
	procSetConsoleTitleW = kernel32.NewProc("SetConsoleTitleW")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
)

const enableVT = 0x0004

var consoleAllocated bool

// ShowConsole allocates a console for a GUI-subsystem process.
// Safe to call multiple times.
func ShowConsole(title string) {
	if consoleAllocated {
		return
	}
	r, _, _ := procAllocConsole.Call()
	if r == 0 {
		return
	}
	consoleAllocated = true

	if out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0); err == nil {
		os.Stdout = out
		os.Stderr = out
	}
	if in, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = in
	}

	if t, err := syscall.UTF16PtrFromString(title); err == nil {
		procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(t)))
	}

	// Enable ANSI / \r progress bars on Windows console.
	h := uintptr(os.Stdout.Fd())
	var mode uint32
	procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	mode |= enableVT
	procSetConsoleMode.Call(h, uintptr(mode))
}

func PauseConsole() {
	if !consoleAllocated {
		return
	}
	fmt.Fprint(os.Stdout, "\nPress Enter to continue...")
	r := bufio.NewReader(os.Stdin)
	_, _ = r.ReadString('\n')
}
