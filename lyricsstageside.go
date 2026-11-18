package main

import (
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/player"
	"github.com/fzn0x/mixmusic/locales"
)

// What the full-screen lyrics (lyricsstage.go) show at their sides: at the
// left of the cover, the buttons that play and pause, and go to the next and
// the previous track; at the right, beside the lyrics' scroll bar, the covers
// of the queue around the track playing, which a click plays.

const (
	// stageControlsW is the width of the column of buttons at the left of the
	// cover.
	stageControlsW = 52
	// stageQueueW is the width of the strip of the queue's covers. The
	// stage keeps it whether or not there are tracks next, so that the lyrics
	// don't change width, and wrap anew, as the queue runs out.
	stageQueueW = 68
	// stageQueueCovers is the most covers the strip shows.
	stageQueueCovers = 10
)

// stageButton is a round button of the stage's ink, size across with a glyph
// of the size given: filled, if main, and outlined by its hover otherwise.
// It tells whether it was pressed. The stage's ink is white, whatever the
// theme, as the ground is dark.
func (s *shell) stageButton(c *ui.Context, key, name, label string, size, glyph float32, main bool) bool {
	b := s.buttonBase(c.Key(key)).Size(size, size).Shrink(0).Radius(size / 2).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	ink := stageInk
	switch {
	case main:
		bg := stageInk
		if b.Hovered() {
			bg = stageInk.Alpha(0.86)
		}
		b.Background(bg)
		ink = ui.Color{A: 255}
	case b.Pressed():
		b.Background(stageInk.Alpha(0.28))
	case b.Hovered():
		b.Background(stageInk.Alpha(0.16))
	}
	s.pressMotion(b, 0.9)
	b.Children(func() { ui.Icon(c, icon(name)).FontSize(glyph).TextColor(ink) })
	return b.Clicked()
}

// stageControls are the buttons at the left of the cover, in a column as tall
// as the cover, so that they lie at its middle: the previous track, play or
// pause, and the next track.
func (s *shell) stageControls(c *ui.Context, cover float32) {
	name, label := toggleOf(s.st)
	ui.Column(c.Key("stage-controls")).Width(stageControlsW).Height(cover).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Gap(14).
		Label(locales.LyricsStageControls()).Children(func() {
		if s.stageButton(c, "stage-previous", "previous", locales.CmdPlayerPrevious(), 40, 18, false) {
			s.p.Previous()
		}
		if s.stageButton(c, "stage-toggle", name, label, 52, 22, true) {
			s.p.Toggle()
		}
		if s.stageButton(c, "stage-next", "next", locales.CmdPlayerNext(), 40, 18, false) {
			s.p.Next()
		}
	})
}

// stageQueue is the strip of the covers of the queue, h high, as many as fit
// between the close button and the foot: the entry playing at a fixed spot,
// a third of the way down, with the entries played before it above and those
// next below it, as the queue panel reads. A click on a cover plays its
// entry. When the entry playing changes, the covers slide to their new
// places, those leaving fading out and those coming fading in (MyGo leaves
// the sliding out while the desktop asks for less motion). The strip takes
// its room even when it is empty.
func (s *shell) stageQueue(c *ui.Context, h float32) {
	const size, gap = 52, 12
	q, at := s.st.Queue, s.st.Index
	room := min(max(int((h-2*80+gap)/(size+gap)), 0), stageQueueCovers)
	blank, from, to := stageQueueWindow(len(q), at, room)
	// The covers start where room of them would, centred, so that the one
	// playing keeps its spot as the queue runs out at either end.
	top := max((h-float32(room)*(size+gap)+gap)/2, 0)
	ui.Column(c.Key("stage-queue")).Width(stageQueueW).Height(h).Shrink(0).AlignItems(ui.Center).Justify(ui.Start).Gap(gap).Padding(top, 0, 0, 0).
		Label(locales.PanelUpNext()).Children(func() {
		for range blank {
			ui.Box(c).Size(size, size).Shrink(0)
		}
		for i := from; i < to; i++ {
			s.stageQueueCover(c, i, q[i], i-at, size)
		}
	})
}

// stageQueueWindow is which of the n entries of a queue the strip shows, room
// covers at most, around the entry at playing: blank empty spots, then the
// entries from from to to (not included), so that the one playing lies at
// the spot room/3, after blank spots and the entries played before it. With
// nothing playing (at -1), it is the queue from its start.
func stageQueueWindow(n, at, room int) (blank, from, to int) {
	if room <= 0 || n == 0 {
		return 0, 0, 0
	}
	if at < 0 || at >= n {
		return 0, 0, min(n, room)
	}
	lead := room / 3
	from = max(at-lead, 0)
	blank = lead - (at - from)
	return blank, from, min(n, from+room-blank)
}

// stageQueueCover is the cover of the entry i of the queue, rank places after
// the one playing (negative before it, 0 for it). The one playing shows
// plain, ringed in the stage's ink, and a click on it does nothing; the
// nearer to playing one next lies, the plainer it shows; those played show
// dimmer still, and an entry the queue passes over shows dim.
func (s *shell) stageQueueCover(c *ui.Context, i int, it player.Item, rank int, size float32) {
	t := it.Track
	b := s.buttonBase(c.Key("stage-q-"+it.UID)).Size(size, size).Shrink(0).Radius(10).Clip().
		Tooltip(t.Title + " · " + names(t.Artists)).
		Transition(ui.ElementTransition{Position: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
	if rank == 0 {
		b.Label(locales.MiniPanelPlaying(t.Title)).Border(2, stageInk)
		b.Children(func() { s.cover(c, "stage-q-cover-"+it.UID, t.Artwork, size, false) })
		return
	}
	b.Label(locales.TracksPlay(t.Title)).Cursor(ui.CursorPointer)
	op := float32(0.3)
	if rank > 0 {
		op = max(1-0.09*float32(rank-1), 0.45)
	}
	if it.Skip != "" {
		op = min(op, 0.35)
	}
	if b.Hovered() {
		op = 1
		b.Border(2, stageInk)
	}
	b.Opacity(op)
	b.Children(func() { s.cover(c, "stage-q-cover-"+it.UID, t.Artwork, size, false) })
	if b.Clicked() {
		s.p.PlayAt(i)
	}
}
