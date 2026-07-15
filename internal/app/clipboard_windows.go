//go:build windows

package app

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	clipboardKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	clipboardUser32   = windows.NewLazySystemDLL("user32.dll")
	globalLockProc    = clipboardKernel32.NewProc("GlobalLock")
	globalUnlockProc  = clipboardKernel32.NewProc("GlobalUnlock")
	globalSizeProc    = clipboardKernel32.NewProc("GlobalSize")
	globalAllocProc   = clipboardKernel32.NewProc("GlobalAlloc")
	globalFreeProc    = clipboardKernel32.NewProc("GlobalFree")
	openClipboard     = clipboardUser32.NewProc("OpenClipboard")
	closeClipboard    = clipboardUser32.NewProc("CloseClipboard")
	emptyClipboard    = clipboardUser32.NewProc("EmptyClipboard")
	getClipboardData  = clipboardUser32.NewProc("GetClipboardData")
	setClipboardData  = clipboardUser32.NewProc("SetClipboardData")
	formatAvailable   = clipboardUser32.NewProc("IsClipboardFormatAvailable")
)

const globalMemMoveable uintptr = 0x0002

func readSystemClipboard() (string, uint, error) {
	formats := []uint{clipboardUnicodeText, clipboardHTML, clipboardRTF}
	var lastErr error
	for _, format := range formats {
		text, err := readClipboardFormat(format)
		if err == nil && text != "" {
			return text, format, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("クリップボードから表を読み込めません")
	}
	return "", 0, lastErr
}

func readClipboardFormat(format uint) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	available, _, _ := formatAvailable.Call(uintptr(format))
	if available == 0 {
		return "", fmt.Errorf("指定のクリップボード形式は存在しません")
	}
	if err := waitOpenSystemClipboard(); err != nil {
		return "", err
	}
	defer closeClipboard.Call()
	handle, _, _ := getClipboardData.Call(uintptr(format))
	if handle == 0 {
		return "", fmt.Errorf("クリップボードデータを取得できません")
	}
	pointer, _, _ := globalLockProc.Call(handle)
	if pointer == 0 {
		return "", fmt.Errorf("クリップボードデータをロックできません")
	}
	defer globalUnlockProc.Call(handle)
	size, _, _ := globalSizeProc.Call(handle)
	if format == clipboardUnicodeText {
		data := (*[1 << 30]uint16)(unsafe.Pointer(pointer))[: size/2 : size/2]
		return syscall.UTF16ToString(data), nil
	}
	data := (*[1 << 30]byte)(unsafe.Pointer(pointer))[:size:size]
	return strings.TrimRight(string(data), "\x00"), nil
}

func waitOpenSystemClipboard() error {
	deadline := time.Now().Add(time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		opened, _, err := openClipboard.Call(0)
		if opened != 0 {
			return nil
		}
		lastErr = err
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("クリップボードを開けません: %w", lastErr)
}

func writeSystemClipboard(text string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := waitOpenSystemClipboard(); err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			closeClipboard.Call()
		}
	}()
	cleared, _, err := emptyClipboard.Call()
	if cleared == 0 {
		return fmt.Errorf("クリップボードを空にできません: %w", err)
	}
	text = strings.ReplaceAll(text, "\x00", "")
	data, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("クリップボード文字列を変換できません: %w", err)
	}
	handle, _, err := globalAllocProc.Call(globalMemMoveable, uintptr(len(data))*2)
	if handle == 0 {
		return fmt.Errorf("クリップボード用メモリを確保できません: %w", err)
	}
	owned := false
	defer func() {
		if !owned {
			globalFreeProc.Call(handle)
		}
	}()
	pointer, _, err := globalLockProc.Call(handle)
	if pointer == 0 {
		return fmt.Errorf("クリップボード用メモリをロックできません: %w", err)
	}
	copy((*[1 << 30]uint16)(unsafe.Pointer(pointer))[:len(data):len(data)], data)
	globalUnlockProc.Call(handle)
	result, _, err := setClipboardData.Call(uintptr(clipboardUnicodeText), handle)
	if result == 0 {
		return fmt.Errorf("クリップボードへデータを設定できません: %w", err)
	}
	owned = true
	if result, _, err := closeClipboard.Call(); result == 0 {
		closed = true
		return fmt.Errorf("クリップボードを閉じられません: %w", err)
	}
	closed = true
	return nil
}
