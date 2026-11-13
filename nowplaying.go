package main

import (
	"context"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
)

// nowPlayingPage gives the track playing the whole pane: its sleeve under
// a visualizer, or the picture of a music video, with its credits under
// it, its lyrics, and what relates to it.
func (s *shell) nowPlayingPage(c *ui.Context) {
	k := &s.lk
	cur, ok := s.st.Current()
	s.pageScroll(c, func() {
		if !ok {
			s.pageHeader(c, locales.NowPlayingTitle(), nil, nil)
			ui.Text(c, k.words(locales.NowPlayingNothing())).FontSize(k.bigSize()).FontWeight(k.displayWeight()).Font(k.display).TextColor(k.accent)
			ui.Text(c, locales.NowPlayingEmpty()).FontSize(k.subtitleSize()).FontWeight(k.leadWeight())
			return
		}
		t := cur.Track
		s.pageHeader(c, t.Title, nil, nil)
		pane := s.paneWidth(c) - 2*k.gutter
		// The sleeve goes above the lyrics only where both can't sit side by side.
		stack := pane < 640
		grid := ui.Row(c.Key("now-grid")).Gap(48).AlignItems(ui.Start)
		if stack {
			grid.Column()
		}
		// Beside the sleeve, the lyrics fill the window's height under the
		// head, however tall it is, and the sleeve grows with the room.
		room := float32(420)
		if top, _, ok := s.downPage(grid); ok && !stack {
			room = max(room, s.here.view-top-32)
		}
		// Bounds are the frame before's: one more frame settles a resize.
		if room != s.lyricsView.room {
			s.lyricsView.room = room
			c.Invalidate()
		}
		sleeve := min(max(260, pane*0.4), max(420, room-48))
		if stack && s.here.view > 0 {
			// Above the lyrics, it leaves them room in the window.
			sleeve = min(sleeve, max(200, s.here.view*0.4))
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
			ui.Column(c).Grow(1).MinWidth(0).Gap(24).Children(func() {
				ui.Column(c).Height(room).Children(func() { s.lyrics(c, true) })
				s.pluginWidgets(c)
			})
		})
		s.related(c, t)
	})
}

// related shows what YouTube Music relates to the track, under the player:
// songs, playlists and artists like it. When it fails, the page goes
// without it.
func (s *shell) related(c *ui.Context, t media.Track) {
	res := fetch(s, "related-"+t.ID, func(ctx context.Context) ([]media.Shelf, error) {
		return s.svc.Catalog.Related(ctx, t)
	})
	if !res.done || res.err != nil || len(res.value) == 0 {
		return
	}
	ui.Column(c).Margin(48, 0, 0).Children(func() { s.shelves(c, "now-related", res.value) })
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
