package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// pageMove is how the page coming in moves, as the router fades it in:
// from where it starts (from), in the first frame of it (starting), until
// it stops (posing).
type pageMove struct {
	path             string // the path shown last
	from             ui.Pose
	posing, starting bool
	frames           int
}

// detailPages are the first parts of the paths of pages one level deeper
// than those the sidebar chooses: they slide in, the others rise.
var detailPages = map[string]bool{"album": true, "artist": true, "playlist": true, "podcast": true, "mood": true, "remote": true}

// pageEntry is where the page at path to starts, coming from the page at
// from: across from the side it sits on going deeper, and back from the
// other, or up a few DIPs from below, between pages of the sidebar. Pages
// of the same part of the app, as the tabs of the library, don't move.
//
// shortcut: back is read from the history having pages ahead, so Forward
// to a page with more ahead slides as back does, until the router says
// which way it went.
func pageEntry(from, to string, back, rtl bool) (ui.Pose, bool) {
	first := func(p string) string { s, _, _ := strings.Cut(strings.TrimPrefix(p, "/"), "/"); return s }
	a, b := first(from), first(to)
	deep := detailPages[b] || b == strings.TrimPrefix(roomsPath, "/") && strings.Count(to, "/") > 1
	switch {
	case !deep && a == b:
		return ui.Pose{}, false
	case deep && !back || detailPages[a] && back:
		x := float32(24)
		if back {
			x = -x
		}
		if rtl {
			x = -x
		}
		return ui.Pose{X: x}, true
	}
	return ui.Pose{Y: 10}, true
}

// followPage starts the page's move as the router shows another path,
// but on the window's first frame and in the themes that move nothing.
func (s *shell) followPage(c *ui.Context) {
	m := &s.move
	path := s.router.Path()
	if path == m.path {
		return
	}
	old := m.path
	m.path = path
	if old == "" || !s.lk.moves() {
		return
	}
	m.from, m.posing = pageEntry(old, path, s.router.CanGoForward(), c.Direction() == ui.RightToLeft)
	m.starting, m.frames = m.posing, 0
}

// movePage poses el, the page shown, as its move has it.
func (s *shell) movePage(c *ui.Context, el ui.Element) {
	m := &s.move
	if !m.posing {
		return
	}
	p := ui.Pose{}
	if m.starting {
		// Posed where it starts in its first frame, it shows there at
		// once, and moves to its place from the next.
		p, m.starting = m.from, false
		c.Invalidate()
	}
	el.AnimatePose(p, s.lk.spring(true))
	if m.frames++; m.frames > 2 && !el.Transitioning() {
		m.posing = false
	}
}

// panelOpen is how far the side panel is open, from 0 to 1, row being
// the row that holds it: it opens on the theme's spring, and closes on a
// quicker curve, a press on the way turning it around. A panel there from
// the window's first frame shows open at once, as one does in the themes
// that move nothing; as a fraction of its width, it follows the window
// resizing at once.
func (s *shell) panelOpen(row ui.Element) float32 {
	target := float32(0)
	if s.panel != "" {
		s.panelShown, target = s.panel, 1
	}
	k := &s.lk
	if !k.moves() {
		return target
	}
	a := k.spring(true)
	if target == 0 {
		a = ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut}
	}
	return row.AnimateTo("panel", target, a)
}

// fadeMotion cross-fades what replaces something in place, as the queue
// and the lyrics in the side panel.
func (s *shell) fadeMotion() ui.ElementTransition {
	k := &s.lk
	if !k.moves() {
		return ui.ElementTransition{Duration: -1}
	}
	a := k.spring(false)
	return ui.ElementTransition{Animation: &a, Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}}
}

// enterStage brings the full-screen lyrics in, rising a few DIPs as they
// grow to their size, and fades them out quicker as they close.
func (s *shell) enterStage(c *ui.Context, back, panel ui.Element) {
	k := &s.lk
	if !k.moves() {
		return
	}
	back.Transition(ui.ElementTransition{Duration: 150 * time.Millisecond, Colors: true, Exit: &ui.Motion{}})
	entered := ui.Local(panel, "entered", func() bool { return false })
	p := ui.Pose{}
	if !*entered {
		p, *entered = ui.Pose{Y: 8, Scale: 0.98, Fade: 1}, true
		c.Invalidate()
	}
	panel.AnimatePose(p, k.spring(true))
}
