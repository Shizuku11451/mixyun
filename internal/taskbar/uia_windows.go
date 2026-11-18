//go:build windows && (amd64 || arm64)

package taskbar

// The taskbar's buttons are XAML on Windows 11, with no windows of their
// own: UI Automation is what tells where they are, as TaskbarLyrics reads
// them, for the lyrics to keep clear of.

import (
	"log"
	"runtime"
	"slices"
	"syscall"
	"time"
	"unsafe"
)

var (
	clsidCUIAutomation = guid{0xFF48DBA4, 0x60EF, 0x4201, [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E}}
	iidIUIAutomation   = guid{0x30CBE57D, 0xD9D0, 0x452A, [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE}}

	ole32              = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	procCoUninitialize = ole32.NewProc("CoUninitialize")
)

const (
	coinitMultithreaded = 0x0

	// The methods called, by their slot in their vtable
	// (UIAutomationClient.h).
	slotElementFromHandle       = 6  // IUIAutomation
	slotCreatePropertyCondition = 23 // IUIAutomation
	slotFindAll                 = 6  // IUIAutomationElement
	slotBoundingRectangle       = 43 // IUIAutomationElement
	slotArrayLength             = 3  // IUIAutomationElementArray
	slotArrayElement            = 4  // IUIAutomationElementArray

	treeScopeDescendants     = 4
	uiaControlTypePropertyID = 30003
	uiaButtonControlTypeID   = 50000
	vtI4                     = 3
)

// variant is a VARIANT, as 64-bit Windows lays it out.
type variant struct {
	vt  uint16
	_   [3]uint16
	val int64
	_   int64
}

// comCall calls the method at slot of the COM object o.
func comCall(o unsafe.Pointer, slot int, args ...uintptr) uintptr {
	m := unsafe.Slice(*(**uintptr)(o), slot+1)[slot]
	r, _, _ := syscall.SyscallN(m, append([]uintptr{uintptr(o)}, args...)...)
	return r
}

// watchButtons measures the primary taskbar's buttons every lyricsReplace,
// for embed to keep the lyrics clear of, until the window closes. On a
// thread of its own: UI Automation waits on Explorer, which the window's
// thread, drawing frames, mustn't.
func (l *lyricsWin) watchButtons() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if procCoInitializeEx.Find() != nil {
		return
	}
	if r, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded); check(r) != nil {
		return
	}
	defer procCoUninitialize.Call()
	var uia unsafe.Pointer
	r, _, _ := syscall.SyscallN(procCoCreateInstance.Addr(), uintptr(unsafe.Pointer(&clsidCUIAutomation)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)), uintptr(unsafe.Pointer(&uia)))
	if check(r) != nil || uia == nil {
		return
	}
	defer comCall(uia, slotRelease)
	for !l.isClosed() {
		// A failed reading keeps the last: the lyrics stay where they are.
		if b, ok := taskbarButtons(uia, findWindow(0, "Shell_TrayWnd")); ok {
			l.mu.Lock()
			changed := !slices.Equal(l.buttons, b)
			l.buttons = b
			l.mu.Unlock()
			if changed {
				log.Printf("taskbar: %d buttons: %v", len(b), b)
				procPostMessageW.Call(l.hwnd, wmLyricsUpdate, 0, 0)
			}
		}
		time.Sleep(lyricsReplace * time.Millisecond)
	}
}

// taskbarButtons is where the buttons in the taskbar bar are on the
// screen: Start, the widgets, search, the apps', the notification area's.
// shortcut: buttons alone; a search box that isn't one goes under the
// lyrics, add its control type if one does.
func taskbarButtons(uia unsafe.Pointer, bar uintptr) ([]rect, bool) {
	if bar == 0 {
		return nil, false
	}
	var el, cond, all unsafe.Pointer
	if check(comCall(uia, slotElementFromHandle, bar, uintptr(unsafe.Pointer(&el)))) != nil || el == nil {
		return nil, false
	}
	defer comCall(el, slotRelease)
	// A VARIANT, wider than a register, goes by a pointer to its copy.
	v := variant{vt: vtI4, val: uiaButtonControlTypeID}
	if check(comCall(uia, slotCreatePropertyCondition, uiaControlTypePropertyID, uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&cond)))) != nil || cond == nil {
		return nil, false
	}
	defer comCall(cond, slotRelease)
	if check(comCall(el, slotFindAll, treeScopeDescendants, uintptr(cond), uintptr(unsafe.Pointer(&all)))) != nil || all == nil {
		return nil, false
	}
	defer comCall(all, slotRelease)
	var n int32
	if check(comCall(all, slotArrayLength, uintptr(unsafe.Pointer(&n)))) != nil {
		return nil, false
	}
	buttons := make([]rect, 0, n)
	for i := range n {
		var b unsafe.Pointer
		if check(comCall(all, slotArrayElement, uintptr(i), uintptr(unsafe.Pointer(&b)))) != nil || b == nil {
			continue
		}
		var r rect
		if check(comCall(b, slotBoundingRectangle, uintptr(unsafe.Pointer(&r)))) == nil {
			buttons = append(buttons, r)
		}
		comCall(b, slotRelease)
	}
	return buttons, true
}
