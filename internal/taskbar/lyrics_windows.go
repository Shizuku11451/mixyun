//go:build windows && (amd64 || arm64)

package taskbar

// The lyrics' window is a layered child of Shell_TrayWnd that this package
// draws itself, with GDI and UpdateLayeredWindow, on a thread of its own.
// A window that draws through DXGI or DirectComposition, as mygo's do,
// shows nothing once it is a child of Explorer's taskbar; a layered one
// does, as TrafficMonitor's and TaskbarLyrics' do. Its thread runs only
// this window: SetParent joins its input to Explorer's, which the app's
// main thread stays apart from.

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	wmClose        = 0x0010
	wmDestroy      = 0x0002
	wmTimer        = 0x0113
	wmNCHitTest    = 0x0084
	wmLyricsUpdate = 0x8000 + 0x4C // WM_APP + 'L'

	wsPopup         = 0x80000000
	wsExLayered     = 0x00080000
	wsExTransparent = 0x00000020
	wsExToolWindow  = 0x00000080
	wsExNoActivate  = 0x08000000

	swHide           = 0
	swShowNoActivate = 4
	htTransparent    = ^uintptr(0) // -1
	ulwAlpha         = 2

	// The font's: FW_BOLD, DEFAULT_CHARSET and ANTIALIASED_QUALITY,
	// grey edges whose level is the glyph's alpha, where ClearType's
	// coloured ones would have none.
	fwBold             = 700
	defaultCharset     = 1
	antialiasedQuality = 4
	transparentBk      = 1
	dtLeft             = 0x0
	dtVCenter          = 0x4
	dtSingleLine       = 0x20
	dtNoPrefix         = 0x800
	dtEndEllipsis      = 0x8000

	// lyricsFace is the lines' face: Chinese, with kana, on every Windows
	// 10 and 11, and Latin.
	lyricsFace = "Microsoft YaHei UI"
	// lyricsReplace is how often, in milliseconds, the window takes its
	// place again, as the notification area grows or the taskbar moves.
	lyricsReplace = 1000
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procShowWindow          = user32.NewProc("ShowWindow")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procSetTimer            = user32.NewProc("SetTimer")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")
	procDrawTextW           = user32.NewProc("DrawTextW")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procCreateFontW        = gdi32.NewProc("CreateFontW")
	procSetTextColor       = gdi32.NewProc("SetTextColor")
	procSetBkMode          = gdi32.NewProc("SetBkMode")
	procGdiFlush           = gdi32.NewProc("GdiFlush")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

// errNoTaskbar is returned where the taskbar takes no window: there is
// none, or it stands at a side.
var errNoTaskbar = errors.New("taskbar: the taskbar takes no window of the lyrics")

type wndClassEx struct {
	size, style         uint32
	wndProc             uintptr
	clsExtra, wndExtra  int32
	instance, icon      uintptr
	cursor, background  uintptr
	menuName, className *uint16
	iconSm              uintptr
}

type bitmapInfoHeader struct {
	size          uint32
	width, height int32
	planes, bits  uint16
	compression   uint32
	sizeImage     uint32
	xppm, yppm    int32
	used, import_ uint32
}

type winPoint struct{ x, y int32 }

type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      winPoint
	private uint32
}

var (
	lyricsClass     *uint16
	lyricsClassErr  error
	lyricsClassOnce sync.Once

	// lyricsWindows are the open windows, by their handle, for the
	// window procedure.
	lyricsMu      sync.Mutex
	lyricsWindows = map[uintptr]*lyricsWin{}
)

// lyricsWin is the window of the lyrics.
type lyricsWin struct {
	width int
	hwnd  uintptr // set before the window procedure sees it; its thread's

	mu     sync.Mutex
	view   LyricsView
	closed bool

	// What the window shows, on its thread: to draw again only on change.
	drawn struct {
		view  LyricsView
		w, h  int32
		dpi   uint32
		light bool
		ok    bool
	}
}

// openLyrics starts the window's thread, and has it create the window and
// put it in the taskbar.
func openLyrics(width int) (*lyricsWin, error) {
	l := &lyricsWin{width: width}
	ready := make(chan error, 1)
	go l.run(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return l, nil
}

// run creates the window and runs its messages until it is destroyed: by
// close, or with the taskbar, as Explorer exits.
func (l *lyricsWin) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	lyricsClassOnce.Do(registerLyricsClass)
	if lyricsClassErr != nil {
		ready <- lyricsClassErr
		return
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(wsExLayered|wsExTransparent|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(lyricsClass)), 0, wsPopup, 0, 0, 1, 1, 0, 0, inst, 0)
	if hwnd == 0 {
		ready <- errors.New("taskbar: the lyrics' window: " + err.Error())
		return
	}
	l.hwnd = hwnd
	lyricsMu.Lock()
	lyricsWindows[hwnd] = l
	lyricsMu.Unlock()
	if !embed(hwnd, l.width) {
		lyricsMu.Lock()
		delete(lyricsWindows, hwnd)
		lyricsMu.Unlock()
		procDestroyWindow.Call(hwnd)
		ready <- errNoTaskbar
		return
	}
	procSetTimer.Call(hwnd, 1, lyricsReplace, 0)
	ready <- nil
	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
}

func registerLyricsClass() {
	lyricsClass, _ = syscall.UTF16PtrFromString("MixMusicTaskbarLyrics")
	inst, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{wndProc: syscall.NewCallback(lyricsProc), instance: inst, className: lyricsClass}
	wc.size = uint32(unsafe.Sizeof(wc))
	if a, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); a == 0 {
		lyricsClassErr = errors.New("taskbar: the lyrics' window class: " + err.Error())
	}
}

func lyricsProc(hwnd, msg, wp, lp uintptr) uintptr {
	lyricsMu.Lock()
	l := lyricsWindows[hwnd]
	lyricsMu.Unlock()
	switch msg {
	case wmLyricsUpdate, wmTimer:
		if l != nil {
			l.update()
		}
		return 0
	case wmNCHitTest:
		// The taskbar under it takes the clicks.
		return htTransparent
	case wmDestroy:
		lyricsMu.Lock()
		delete(lyricsWindows, hwnd)
		lyricsMu.Unlock()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

// show has the window's thread show v, or hide without v.Current.
func (l *lyricsWin) show(v LyricsView) {
	l.mu.Lock()
	l.view = v
	closed := l.closed
	l.mu.Unlock()
	if !closed {
		procPostMessageW.Call(l.hwnd, wmLyricsUpdate, 0, 0)
	}
}

// close destroys the window, from any goroutine.
func (l *lyricsWin) close() {
	l.mu.Lock()
	closed := l.closed
	l.closed = true
	l.mu.Unlock()
	if !closed {
		procPostMessageW.Call(l.hwnd, wmClose, 0, 0)
	}
}

// isClosed tells whether the window is gone, as with Explorer.
func (l *lyricsWin) isClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed
}

// update puts the window in its place in the taskbar again, and draws the
// view there, or hides it. The window's thread.
func (l *lyricsWin) update() {
	l.mu.Lock()
	v := l.view
	l.mu.Unlock()
	visible, _, _ := procIsWindowVisible.Call(l.hwnd)
	if v.Current == "" || !embed(l.hwnd, l.width) {
		if visible != 0 {
			procShowWindow.Call(l.hwnd, swHide)
		}
		l.drawn.ok = false
		return
	}
	r, ok := windowRect(l.hwnd)
	w, h := r.right-r.left, r.bottom-r.top
	if !ok || w <= 0 || h <= 0 {
		return
	}
	dpi := uint32(96)
	if procGetDpiForWindow.Find() == nil {
		if d, _, _ := procGetDpiForWindow.Call(l.hwnd); d != 0 {
			dpi = uint32(d)
		}
	}
	light := taskbarLight()
	d := &l.drawn
	if !d.ok || d.view != v || d.w != w || d.h != h || d.light != light || d.dpi != dpi {
		if !l.draw(v, w, h, dpi, light) {
			return
		}
		d.view, d.w, d.h, d.light, d.dpi, d.ok = v, w, h, light, dpi, true
	}
	if visible == 0 {
		procShowWindow.Call(l.hwnd, swShowNoActivate)
	}
}

// draw draws v in the window w by h pixels. GDI draws the lines white on
// black, the next one grey, whose level is then their alpha in the
// taskbar's ink; the cover goes over that at the left.
func (l *lyricsWin) draw(v LyricsView, w, h int32, dpi uint32, light bool) bool {
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	dc, _, _ := procCreateCompatibleDC.Call(screen)
	if dc == 0 {
		return false
	}
	defer procDeleteDC.Call(dc)
	bi := bitmapInfoHeader{width: w, height: -h, planes: 1, bits: 32} // top-down
	bi.size = uint32(unsafe.Sizeof(bi))
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return false
	}
	defer procDeleteObject.Call(bmp)
	oldBmp, _, _ := procSelectObject.Call(dc, bmp)
	defer procSelectObject.Call(dc, oldBmp)

	lay := layoutView(w, h, dpi, v.Cover != nil)
	procSetBkMode.Call(dc, transparentBk)
	current := lay.current
	if v.Next == "" {
		// Alone, the line sits in the middle.
		current.top, current.bottom = 0, h
	}
	drawLine(dc, v.Current, current, lay.currentPx, viewCurrentAlpha)
	if v.Next != "" {
		drawLine(dc, v.Next, lay.next, lay.nextPx, viewNextAlpha)
	}
	procGdiFlush.Call()

	ink := [3]uint8{255, 255, 255}
	if light {
		ink = [3]uint8{0x1f, 0x1f, 0x1f}
	}
	px := unsafe.Slice((*uint32)(bits), int(w)*int(h))
	inkPixels(px, ink)
	if v.Cover != nil && !lay.cover.Empty() {
		pasteCover(px, int(w), lay.cover.Min, roundCover(v.Cover, lay.cover.Dx(), lay.radius))
	}

	size, src := winPoint{w, h}, winPoint{}
	blend := uint32(0xFF<<16 | 1<<24) // AC_SRC_OVER, opaque, AC_SRC_ALPHA
	ok, _, _ := procUpdateLayeredWindow.Call(l.hwnd, screen, 0, uintptr(unsafe.Pointer(&size)), dc,
		uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)
	return ok != 0
}

// drawLine draws text in box, px tall, grey at alpha of white, cut with an
// ellipsis where it doesn't fit.
func drawLine(dc uintptr, text string, box rect, px int32, alpha float64) {
	face, _ := syscall.UTF16PtrFromString(lyricsFace)
	font, _, _ := procCreateFontW.Call(uintptr(-px), 0, 0, 0, fwBold, 0, 0, 0, defaultCharset, 0, 0,
		antialiasedQuality, 0, uintptr(unsafe.Pointer(face)))
	if font != 0 {
		defer procDeleteObject.Call(font)
		old, _, _ := procSelectObject.Call(dc, font)
		defer procSelectObject.Call(dc, old)
	}
	g := uintptr(alpha*255 + 0.5)
	procSetTextColor.Call(dc, g|g<<8|g<<16)
	s, _ := syscall.UTF16FromString(text)
	procDrawTextW.Call(dc, uintptr(unsafe.Pointer(&s[0])), uintptr(len(s)-1), uintptr(unsafe.Pointer(&box)),
		dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
}
