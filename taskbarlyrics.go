package main

// The window over Windows' taskbar that shows the line of the lyrics sung
// now, while Status bar lyrics is on: the app opens and redraws it
// (internal/app/statuslyrics.go). macOS shows the line in its menu bar,
// beside the tray icon, without a window.

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/internal/taskbar"
	"github.com/fzn0x/mixmusic/locales"
)

// taskbarLyricsWidth is the window's width, and taskbarTrayRoom what it
// leaves at the taskbar's right for the notification area and the clock.
const (
	taskbarLyricsWidth = 360
	taskbarTrayRoom    = 320
)

// OpenTaskbarLyrics opens the window in Windows' taskbar, its child, as
// TaskbarLyrics has it, or over it where the taskbar takes no child; it
// returns nil elsewhere, and where the taskbar stands at a side.
func OpenTaskbarLyrics(a *app.App) *mygo.Window {
	if runtime.GOOS != "windows" {
		return nil
	}
	r, over := taskbarLyricsPlace(mygo.Screen.Displays())
	if !over {
		r = mygo.Rectangle{Width: taskbarLyricsWidth, Height: 40}
	}
	registerFonts()
	var win *mygo.Window
	var embedded bool
	view := func(c *ui.Context) {
		ink := ui.RGB(255, 255, 255)
		if taskbar.Light() {
			ink = ui.RGB(0, 0, 0)
		}
		ui.Row(c.Key("taskbar-lyrics")).Fill().AlignItems(ui.Center).Justify(ui.End).PaddingX(8).Children(func() {
			ui.Text(c, a.StatusLyric()).FontSize(14).FontWeight(600).TextColor(ink).SingleLine().MinWidth(0)
		})
		if embedded {
			// Again, once drawn, as the notification area grows or the
			// taskbar moves.
			mygo.App.WhenReady(func() { taskbar.Embed(win.NativeHandle(), taskbarLyricsWidth) })
		} else {
			// The taskbar comes over it as it is clicked.
			taskbar.KeepAbove(win.NativeHandle())
		}
	}
	win = mygo.NewWindow(mygo.WindowOptions{
		Title:         locales.PlayerStatusBarLyrics(),
		X:             r.X,
		Y:             r.Y,
		Width:         r.Width,
		Height:        r.Height,
		Hidden:        true,
		Frameless:     true,
		Transparent:   true,
		AlwaysOnTop:   true,
		SkipTaskbar:   true,
		DisableResize: true,
		DisableMove:   true,
		DisableShadow: true,
		Content:       ui.View(view),
	})
	embedded = taskbar.Embed(win.NativeHandle(), taskbarLyricsWidth)
	if !embedded && !over {
		win.Close()
		return nil
	}
	win.ShowInactive()
	return win
}

// taskbarLyricsPlace is where the window goes on the primary display where
// the taskbar takes no child: over it, at the bottom or the top, left of
// the notification area.
// shortcut: the room for the notification area is a guess, not measured,
// as taskbar.Embed measures it; that is only the fallback.
func taskbarLyricsPlace(displays []mygo.Display) (mygo.Rectangle, bool) {
	for _, d := range displays {
		if !d.Primary {
			continue
		}
		b, w := d.Bounds, d.WorkArea
		r := mygo.Rectangle{X: b.X + b.Width - taskbarTrayRoom - taskbarLyricsWidth, Width: taskbarLyricsWidth}
		switch {
		case w.Y == b.Y && w.Height < b.Height:
			r.Y, r.Height = w.Y+w.Height, b.Height-w.Height
		case w.Y > b.Y:
			r.Y, r.Height = b.Y, w.Y-b.Y
		default:
			return mygo.Rectangle{}, false
		}
		return r, r.X > b.X
	}
	return mygo.Rectangle{}, false
}
