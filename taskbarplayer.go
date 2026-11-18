package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/locales"
)

// The player the cover of the taskbar's lyrics expands, on Windows: a
// panel over the taskbar's lyrics, with the
// cover, the title, the artists and the service, the transport, and the
// progress, which seeks. It closes as it loses the keyboard, as a flyout
// of the taskbar does, on Escape, and on another click on the cover.

const (
	taskbarPlayerWidth  = 400
	taskbarPlayerHeight = 156
	// taskbarPlayerGap is how far it stays from the taskbar and the
	// screen's edge.
	taskbarPlayerGap = 12
	taskbarPlayerArt = 84
)

// OpenTaskbarPlayer opens the player above the primary display's taskbar,
// over the middle of its lyrics, or at its left end without them.
func OpenTaskbarPlayer(a *app.App) *mygo.Window {
	registerFonts()
	s := newShell(a, a.GoPlayer())
	d := mygo.Screen.PrimaryDisplay()
	work := d.WorkArea
	x := work.X + taskbarPlayerGap
	if r := a.TaskbarLyricsBounds(); !r.Empty() && d.ScaleFactor > 0 {
		// The lyrics' pixels in the DIPs of the primary display, whose
		// pixels start at the screen's origin.
		mid := d.Bounds.X + int(float64(r.Min.X+r.Max.X)/2/d.ScaleFactor)
		x = min(max(mid-taskbarPlayerWidth/2, work.X+taskbarPlayerGap), work.X+work.Width-taskbarPlayerWidth-taskbarPlayerGap)
	}
	s.attach(mygo.NewWindow(mygo.WindowOptions{
		Title:           locales.MiniWindowTitle(),
		X:               x,
		Y:               work.Y + work.Height - taskbarPlayerHeight - taskbarPlayerGap,
		Width:           taskbarPlayerWidth,
		Height:          taskbarPlayerHeight,
		UseContentSize:  true,
		Frameless:       true,
		AlwaysOnTop:     true,
		SkipTaskbar:     true,
		CornerRadius:    8,
		DisableResize:   true,
		DisableMove:     true,
		DisableMinimize: true,
		DisableMaximize: true,
		BackgroundColor: hexColor(s.lk.bg),
		Content:         ui.View(s.taskbarPlayerView),
	}))
	s.follow(s.win.OnBlur(func() { mygo.App.WhenReady(func() { s.win.Close() }) }))
	s.win.Focus()
	return s.win
}

// taskbarPlayerView builds the player above the taskbar.
func (s *shell) taskbarPlayerView(c *ui.Context) {
	s.st = s.p.State()
	s.followArtwork()
	c.SetTheme(&s.lk.theme)
	if c.Shortcut(0, ui.KeyEscape) {
		go s.win.Close()
	}
	k := &s.lk
	cur, ok := s.st.Current()
	ui.Column(c).Fill().Background(k.bg).TextColor(k.text).Font(k.body).Padding(16).Gap(10).Children(func() {
		ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
			art := ui.Box(c.Key("art")).Size(taskbarPlayerArt, taskbarPlayerArt).Shrink(0).Radius(8).Clip()
			var b *ui.Bitmap
			if ok {
				b = s.art.get(c, s.artAt(cur.Track.Artwork, taskbarPlayerArt), s.artPixels(taskbarPlayerArt))
			}
			if b == nil {
				art.Background(k.surface3)
			}
			art.Children(func() {
				if b != nil {
					ui.Image(c, b).Size(taskbarPlayerArt, taskbarPlayerArt).Fit(ui.Cover)
				}
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				title, line := k.words(locales.MiniRowNothingPlaying()), locales.MiniRowPlaySomething()
				if ok {
					title, line = cur.Track.Title, names(cur.Track.Artists)
				}
				ui.Text(c, title).FontSize(k.size(1.15)).FontWeight(k.titleWeight()).Font(k.display).SingleLine()
				ui.Text(c, line).FontSize(k.smallSize()).TextColor(k.muted).SingleLine()
				if ok {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						s.sourceMark(c, cur.Track.Source, false)
						ui.Text(c, sourceName(cur.Track.Source)).FontSize(k.captionSize()).TextColor(k.muted).SingleLine()
					})
				}
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				if s.roundButton(c, "previous", "previous", locales.PlayerPrevious(), false, !ok).Clicked() {
					s.p.Previous()
				}
				name, label := toggleOf(s.st)
				if s.roundButton(c, "toggle", name, label, true, s.st.Loading || !ok && len(s.st.Queue) == 0).Clicked() {
					s.p.Toggle()
				}
				if s.roundButton(c, "next", "next", locales.PlayerNext(), false, !ok).Clicked() {
					s.p.Next()
				}
			})
		})
		s.progress(c)
	})
}
