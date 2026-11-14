package main

import (
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/locales"
)

// nowPlayingPage gives the track playing the whole pane: its sleeve under
// a visualizer, or the picture of a music video, with its credits under
// it, and its lyrics.
func (s *shell) nowPlayingPage(c *ui.Context) {
	k := &s.lk
	cur, ok := s.st.Current()
	if !ok {
		s.pageScroll(c, func() {
			s.pageHeader(c, locales.NowPlayingTitle(), nil, nil)
			ui.Text(c, k.words(locales.NowPlayingNothing())).FontSize(k.bigSize()).FontWeight(k.displayWeight()).Font(k.display).TextColor(k.accent)
			ui.Text(c, locales.NowPlayingEmpty()).FontSize(k.subtitleSize()).FontWeight(k.leadWeight())
		})
		return
	}
	// The page fits the pane and doesn't scroll: only its lyrics do.
	ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Padding(8, k.gutter, 24).Gap(24).Children(func() {
		t := cur.Track
		ui.Text(c, t.Title).FontSize(k.titleSize()).FontWeight(k.displayWeight()).Font(k.display).
			LetterSpacing(k.tracking(-0.02, k.titleSize())).TextColor(k.text).SingleLine().MinWidth(0).Shrink(0)
		pane := s.paneWidth(c) - 2*k.gutter
		// The sleeve goes above the lyrics only where both can't sit side by side.
		stack := pane < 640
		grid := ui.Row(c.Key("now-grid")).Gap(48).AlignItems(ui.Start).Grow(1).MinHeight(0)
		if stack {
			grid.Column().AlignItems(ui.Stretch)
		}
		// The grid takes the height under the title; the sleeve grows with
		// it, leaving room for the credits under it. Bounds are the frame
		// before's: one more frame settles a resize.
		room := grid.LayoutBounds().H
		if room <= 0 {
			room = 300 // the first frame has no bounds yet
		}
		if room != s.lyricsView.room {
			s.lyricsView.room = room
			c.Invalidate()
		}
		sleeve := max(0, min(max(200, pane*0.4), room-48))
		if stack {
			// Above the lyrics, it leaves them room.
			sleeve = min(pane, max(160, room*0.4))
		}
		// The transport is the player bar's: the pane keeps the lyrics room.
		grid.Children(func() {
			ui.Column(c).Width(sleeve).Shrink(0).Gap(16).Children(func() {
				s.sleeve(c, t, sleeve)
				ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
					s.sourceMark(c, t.Source, false)
					s.artistLinks(c, "now-artists", t.Artists, k.text)
					if t.Album != nil && t.Album.Name != "" {
						ui.Text(c, "·").FontSize(k.smallSize()).TextColor(k.muted)
						if t.Album.ID == "" {
							ui.Text(c, t.Album.Name).FontSize(k.smallSize()).TextColor(k.muted)
						} else {
							link := s.buttonBase(c.Key("now-album")).Label(t.Album.Name).Cursor(ui.CursorPointer)
							if link.Clicked() {
								s.navigate(albumPath(t.Album.ID))
							}
							link.Children(func() { ui.Text(c, t.Album.Name).FontSize(k.smallSize()).TextColor(k.muted) })
						}
					}
					if s.st.Waiting {
						ui.Text(c, "·").FontSize(k.smallSize()).TextColor(k.muted)
						ui.Text(c, k.sentence(locales.NowPlayingWaiting())).FontSize(k.smallSize()).TextColor(k.muted)
					}
					s.videoToggle(c)
				})
			})
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).AlignSelf(ui.Stretch).Gap(24).Children(func() {
				ui.Column(c).Grow(1).MinHeight(0).Children(func() { s.lyrics(c, true) })
				s.pluginWidgets(c)
			})
		})
	})
}

// visualizer draws the spectrum of what plays over the foot of the sleeve,
// in the accent, at every frame while it plays. Where MixMusic hears
// nothing, as of a Spotify device elsewhere, it draws nothing.
func (s *shell) visualizer(c *ui.Context, size float32) {
	k := &s.lk
	ui.Box(c).Absolute().Left(0).Bottom(0).Size(size, size/3).PassThrough().Label(locales.NowPlayingVisualizer()).Draw(func(p *ui.Painter, r ui.Rect) {
		p.AnimationFrame()
		bins := s.svc.Player.NativeAnalyser(false, false).Bytes
		const bars = 48
		if len(bins) < bars {
			return
		}
		w := r.W / bars
		for i := range bars {
			// The lower bins carry the music: spread them over the bars.
			v := float32(bins[i*len(bins)/(bars*2)]) / 255
			if v == 0 {
				// Silence: a flat line of bars would look broken.
				continue
			}
			h := max(2, v*r.H)
			p.Fill(ui.Rect{X: r.X + float32(i)*w + 1, Y: r.Y + r.H - h, W: w - 2, H: h}, k.accent.Alpha(0.85), 0)
		}
	})
}
