package main

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/library"
	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
)

var statsRanges = []struct {
	label func() string
	days  int
}{{locales.StatsThisWeek, 7}, {locales.StatsThisMonth, 30}, {locales.StatsAllTime, 0}}

// statsPage shows what the user listened to: artists rank as circles on
// one fixed size ramp, and each service's share shows as two rings.
func (s *shell) statsPage(c *ui.Context) {
	k := &s.lk
	labels := make([]string, len(statsRanges))
	for i, r := range statsRanges {
		labels[i] = r.label()
	}
	years := s.recapYears()
	if len(years) > 0 {
		labels = append(labels, recapLabel())
	}
	if s.statsRange >= len(labels) {
		s.statsRange = 0
	}
	s.pageScroll(c, func() {
		s.pageHeader(c, locales.StatsTitle(), labels, &s.statsRange)
		if len(years) > 0 && s.statsRange == len(statsRanges) {
			s.recapView(c, years)
			return
		}
		st := s.svc.Collection.Stats(statsRanges[s.statsRange].days)
		if st.Plays == 0 {
			ui.Text(c, k.words(locales.StatsNothingCounted())).FontSize(k.bigSize()).FontWeight(k.displayWeight()).Font(k.display).TextColor(k.accent)
			ui.Text(c, locales.StatsCountRule()).FontSize(k.subtitleSize()).FontWeight(k.leadWeight())
			return
		}
		hours := st.Minutes / 60
		spent := locales.FormatMinutes(strconv.Itoa(st.Minutes))
		if hours > 0 {
			spent = locales.FormatHourMinutes(strconv.Itoa(hours), strconv.Itoa(st.Minutes%60))
		}
		size := k.subtitleSize()
		ui.RichText(c,
			ui.Span{Text: locales.StatsYouPlayed(), Size: size, Weight: 300},
			ui.Span{Text: locales.StatsPlayedTracks(st.Plays), Size: size, Weight: 400},
			ui.Span{Text: locales.StatsFor(), Size: size, Weight: 300},
			ui.Span{Text: spent, Size: size, Weight: 400},
			ui.Span{Text: locales.StatsColon(), Size: size, Weight: 300},
			ui.Span{Text: locales.StatsFromSpotify(strconv.Itoa(st.Spotify)), Size: size, Weight: 300, Color: k.spotify},
			ui.Span{Text: locales.StatsComma(), Size: size, Weight: 300},
			ui.Span{Text: locales.StatsFromNetEase(strconv.Itoa(st.NetEase)), Size: size, Weight: 300, Color: k.accent},
			ui.Span{Text: locales.StatsAnd(), Size: size, Weight: 300},
			ui.Span{Text: locales.StatsFromYouTube(strconv.Itoa(st.YouTube)), Size: size, Weight: 300, Color: k.youtube},
			ui.Span{Text: locales.StatsEnd(), Size: size, Weight: 300},
		).MaxWidth(760).Margin(0, 0, 48)
		s.dayChart(c, st.Daily)
		s.heatmap(c)
		ui.Row(c).Gap(48).Wrap().AlignItems(ui.Start).Margin(0, 0, 48).Children(func() {
			ui.Column(c).Width(320).Children(func() {
				s.sectionTitle(c, locales.StatsTheMix())
				s.venn(c, st.Spotify, st.NetEase, st.YouTube)
			})
			ui.Column(c).Grow(1).Basis(320).Children(func() {
				s.sectionTitle(c, locales.StatsTopArtists())
				s.artistCircles(c, st.TopArtists)
			})
		})
		ui.Row(c).Gap(48).Wrap().AlignItems(ui.Start).Margin(0, 0, 48).Children(func() {
			ui.Column(c).Grow(1).Basis(320).Children(func() {
				s.sectionTitle(c, locales.StatsByLanguage())
				s.labelBars(c, st.Languages)
			})
			ui.Column(c).Grow(1).Basis(320).Children(func() {
				s.sectionTitle(c, locales.StatsByGenre())
				ui.Text(c, locales.StatsGenreNote()).FontSize(k.smallSize()).TextColor(k.muted).Margin(0, 0, 12)
				s.labelBars(c, st.Genres)
			})
		})
		s.sectionTitle(c, locales.StatsTopTracks())
		tracks := make([]media.Track, len(st.TopTracks))
		for i, t := range st.TopTracks {
			tracks[i] = t.Track
		}
		s.trackList(c, "top-tracks", tracks, trackListOptions{art: true, album: true})
	})
}

// niceCeil rounds a value up to a clean tick: 1, 2, 2.5 or 5 times a
// power of ten.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if m*p >= v {
			return m * p
		}
	}
	return 10 * p
}

// dayTip is the day of the chart whose minutes show above it: the one the
// pointer came on, or the arrow keys went to.
type dayTip struct {
	day            int
	shown, pointer bool
	// over is the column under the pointer, from 1, and focused whether
	// the chart had the focus, in the last frame; told asks to read the
	// day out, as the keys went to it.
	over          int
	focused, told bool
}

// dayLabel writes a date of the stats as "Tue 20 Oct".
func dayLabel(date string) string {
	t, err := time.ParseInLocation(time.DateOnly, date, time.Local)
	if err != nil {
		return date
	}
	return locales.FormatWeekdayDayMonth(weekdayName(t), strconv.Itoa(t.Day()), monthName(t))
}

// describeDay tells a day's minutes and plays.
func describeDay(d library.DayPlays) string {
	return locales.StatsDayTip(dayLabel(d.Date), strconv.Itoa(d.Minutes), d.Plays)
}

// dayChart draws the minutes listened each day as columns in the accent. A
// day's minutes and plays show above it as the pointer comes on it, or the
// arrow keys go to it while the chart has the focus; below, the days show
// as a table, newest first.
func (s *shell) dayChart(c *ui.Context, days []library.DayPlays) {
	k := &s.lk
	n := len(days)
	if n == 0 {
		return
	}
	tip := &s.statsDay
	tip.day = min(tip.day, n-1)
	peak, most := 0, 0
	for i, d := range days {
		if d.Minutes > most {
			peak, most = i, d.Minutes
		}
	}
	top := niceCeil(float64(most))
	every := 1
	switch {
	case n > 31:
		every = 15
	case n > 7:
		every = 7
	}
	s.sectionTitle(c, locales.StatsDayByDay())
	ui.Text(c, locales.StatsMinutesEachDay()).FontSize(k.smallSize()).TextColor(k.muted).Margin(0, 0, 12)
	ui.Text(c, locales.FormatMinutes(fmt.Sprintf("%.0f", top))).FontSize(k.captionSize()).TextColor(k.muted).FontFeatures("tnum")
	// The chart grows with a wide pane.
	plot := ui.Box(c).Height(min(max(160, 0.12*s.paneWidth(c)), 320)).FillWidth().Margin(4, 0, 0).BorderWidth(0, 0, 1, 0).BorderColor(k.border).
		Label(locales.StatsMinutesLabel()).Description(locales.StatsArrowKeys()).Focusable().
		Draw(func(p *ui.Painter, r ui.Rect) {
			w := r.W / float32(n)
			for i, d := range days {
				h := float32(0)
				if top > 0 {
					h = float32(float64(d.Minutes)/top) * r.H
				}
				if h > 0 {
					fill := k.accent
					if tip.shown && i == tip.day {
						fill = k.accent.Mix(k.text, 0.28)
					}
					p.Fill(ui.Rect{X: r.X + float32(i)*w + max(1, w*0.15), Y: r.Y + r.H - h, W: max(1, w*0.7), H: h}, fill, 0)
				}
				if i == peak && d.Minutes > 0 {
					p.Text(r.X+float32(i)*w, r.Y+r.H-h-16, locales.FormatMinutes(strconv.Itoa(d.Minutes)), k.captionSize(), k.text)
				}
			}
		})
	// The chart's own shortcuts, which go before the window's arrows.
	for _, nav := range []struct {
		key  ui.Key
		step int
	}{{ui.KeyLeft, -1}, {ui.KeyRight, 1}, {ui.KeyHome, -n}, {ui.KeyEnd, n}} {
		if plot.Shortcut(0, nav.key) {
			if !tip.shown {
				tip.day = n - 1
			}
			tip.day, tip.shown, tip.pointer, tip.told = min(max(tip.day+nav.step, 0), n-1), true, false, true
		}
	}
	// A column of the plot for each day, which the pointer comes on and
	// the tip goes above.
	columns := make([]ui.Element, n)
	over := 0
	plot.Children(func() {
		ui.Row(c).Absolute().Top(0).Right(0).Bottom(0).Left(0).AlignItems(ui.Stretch).Children(func() {
			for i := range days {
				columns[i] = ui.Box(c).Grow(1).Basis(0).MinWidth(0)
				if columns[i].Hovered() {
					over = i + 1
				}
			}
		})
	})
	if over != tip.over {
		switch {
		case over > 0:
			tip.day, tip.shown, tip.pointer = over-1, true, true
		case tip.pointer:
			tip.shown, tip.pointer = false, false
		}
		tip.over = over
	}
	switch focused := plot.Focused(); {
	case focused && !tip.focused && !tip.shown:
		tip.day, tip.shown = n-1, true
	case !focused && tip.focused && !tip.pointer:
		tip.shown = false
	}
	tip.focused = plot.Focused()
	if tip.shown {
		told := describeDay(days[tip.day])
		if tip.told {
			c.Announce(told)
			tip.told = false
		}
		// The tip of a day at an end of the chart goes along it, not past.
		at, self := ui.AnchorTop, ui.AnchorBottom
		switch f := (float32(tip.day) + 0.5) / float32(n); {
		case f < 0.1:
			at, self = ui.AnchorTopLeft, ui.AnchorBottomLeft
		case f > 0.9:
			at, self = ui.AnchorTopRight, ui.AnchorBottomRight
		}
		ui.Overlay(c, func() {
			ui.Box(c).AttachTo(columns[tip.day], at, self).MarginY(6).Padding(6, 10).Radius(k.radiusSm).
				Background(k.text).TextColor(k.bg).PassThrough().Children(func() {
				ui.Text(c, told).FontSize(k.smallSize()).FontFeatures("tnum").NoWrap()
			})
		})
	}
	ui.Row(c).FillWidth().Margin(4, 0, 12).Children(func() {
		for i, d := range days {
			label := ""
			switch {
			case i == n-1:
				label = locales.StatsToday()
			case (n-1-i)%every == 0:
				if t, err := time.ParseInLocation("2006-01-02", d.Date, time.Local); err == nil {
					if n <= 7 {
						label = weekdayName(t)
					} else {
						label = locales.FormatDayMonth(strconv.Itoa(t.Day()), monthName(t))
					}
				}
			}
			ui.Text(c, label).FontSize(k.captionSize()).TextColor(k.muted).Grow(1).Basis(0).MinWidth(0).NoWrap()
		}
	})
	// MyGo's Collapsible, built of its parts for pointerKeys to reach its
	// trigger: a click on it leaves Space to play or pause.
	ui.Column(c).Shrink(0).Margin(0, 0, 48).Children(func() {
		t := &k.theme
		table := ui.CollapsibleBase(c, &s.statsTable)
		trigger := s.pointerKeys(table.Trigger).AlignSelf(ui.Start).AlignItems(ui.Center).Gap(t.Space(1)).
			Padding(t.Space(1), 0).PaddingStart(t.Space(0.5)).PaddingEnd(t.Space(2)).Radius(t.Radius)
		if trigger.Hovered() {
			trigger.Background(t.SurfaceHover)
		}
		trigger.Children(func() {
			disclosure(c, 90*table.Progress(), t.TextMuted)
			ui.Text(c, k.words(locales.StatsShowTable())).SingleLine()
		})
		table.Panel(func() {
			columns := []ui.TableColumn{{Title: locales.StatsColDay()}, {Title: locales.StatsColMinutes(), Width: 110, Align: ui.End}, {Title: locales.StatsColPlays(), Width: 90, Align: ui.End}}
			// Ten rows show at once, under the header.
			row := k.theme.Space(8)
			ui.Table(c, &s.statsRows, columns, n, func(i, col int) {
				d := days[n-1-i]
				switch col {
				case 0:
					ui.Text(c, dayLabel(d.Date)).SingleLine()
				case 1:
					ui.Text(c, fmt.Sprint(d.Minutes)).FontFeatures("tnum")
				case 2:
					ui.Text(c, fmt.Sprint(d.Plays)).FontFeatures("tnum")
				}
			}).Height(row*float32(min(n, 10)+1)+1).MaxWidth(520).Margin(8, 0, 0)
		}).Padding(t.Space(1), 0).PaddingStart(t.Space(5.5)).Gap(t.Space(2))
	})
}

// disclosure draws the arrow of a disclosure as MyGo's Collapsible draws
// its own: pointing right, turned by deg degrees clockwise.
func disclosure(c *ui.Context, deg float32, ink ui.Color) {
	ui.Box(c).Size(16, 16).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, d := r.X+r.W/2, r.Y+r.H/2, r.W/8
		sin, cos := math.Sincos(float64(deg) * math.Pi / 180)
		at := func(x, y float32) (float32, float32) {
			return cx + x*float32(cos) - y*float32(sin), cy + x*float32(sin) + y*float32(cos)
		}
		var path ui.Path
		path.MoveTo(at(-d, -2*d)).LineTo(at(d, 0)).LineTo(at(-d, 2*d))
		p.StrokePath(&path, 1.5, ink)
	})
}

// venn draws the plays of each service as rings whose areas match,
// overlapping as the icon's, over a key that names each ring and its
// count: counts in the rings would collide, and the overlap alone would
// read as plays the two share.
func (s *shell) venn(c *ui.Context, spotify, netease, youtube int) {
	k := &s.lk
	rings := []struct {
		source media.Source
		n      int
		ink    ui.Color
		// dx and dy place the ring's centre from the box's, as a corner of
		// a triangle.
		dx, dy float32
	}{
		{media.SourceSpotify, spotify, k.spotify, -38, 22},
		{media.SourceNetEase, netease, k.accent, 38, 22},
		{media.SourceYouTube, youtube, k.youtube, 0, -30},
	}
	most := float64(max(spotify, netease, youtube, 1))
	ui.Box(c).Size(300, 200).Label(locales.StatsVennLabel(strconv.Itoa(spotify), strconv.Itoa(netease), strconv.Itoa(youtube))).
		Draw(func(p *ui.Painter, r ui.Rect) {
			for _, e := range rings {
				var a ui.Path
				a.Circle(r.X+150+e.dx, r.Y+100+e.dy, float32(16+52*math.Sqrt(float64(e.n)/most)))
				p.StrokePath(&a, 2, e.ink)
			}
		})
	ui.Row(c).Gap(24).Wrap().Justify(ui.Center).Width(300).Children(func() {
		for _, e := range rings {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				s.sourceMark(c, e.source, false)
				ui.Text(c, sourceName(e.source)).FontSize(k.smallSize()).TextColor(k.muted)
				ui.Text(c, strconv.Itoa(e.n)).FontSize(k.bodySize())
			})
		}
	})
}

// artistCircles ranks the top artists as circles whose areas grow with
// their plays.
func (s *shell) artistCircles(c *ui.Context, artists []library.ArtistPlays) {
	k := &s.lk
	if len(artists) == 0 {
		return
	}
	top := float64(max(artists[0].Plays, 1))
	ui.Row(c).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		for i, a := range artists {
			r := float32(14 + 46*math.Sqrt(float64(a.Plays)/top))
			// The artist's picture, from their page, which the artist's
			// page itself shares.
			var art []media.Image
			if a.Artist.ID != "" {
				id := a.Artist.ID
				if res := fetch(s, "artist:"+id, func(ctx context.Context) (media.Artist, error) { return s.svc.Catalog.Artist(ctx, id) }); res.done {
					art = res.value.Artwork
				}
			}
			b := s.buttonBase(c.Key("stat-artist-"+count(i))).Size(2*r, 2*r).Radius(r).Background(k.tile(i)).Column().Center().
				Padding(4).Label(locales.StatsArtistPlays(a.Artist.Name, a.Plays)).Tooltip(a.Artist.Name).
				Disabled(a.Artist.ID == "")
			if a.Artist.ID != "" {
				b.Cursor(ui.CursorPointer)
				if b.Clicked() {
					s.navigate(artistPath(a.Artist.ID))
				}
			}
			if len(art) > 0 {
				b.Padding(0).Tooltip(locales.StatsArtistPlays(a.Artist.Name, a.Plays)).Children(func() {
					s.cover(c, "stat-artist-art-"+count(i), art, 2*r, true)
				})
				continue
			}
			b.Children(func() {
				if r >= 30 {
					ui.Text(c, a.Artist.Name).FontSize(k.captionSize()).TextColor(k.tileText).SingleLine().MaxWidth(2*r - 12)
				}
				ui.Text(c, fmt.Sprint(a.Plays)).FontSize(k.captionSize()).TextColor(k.tileText).FontFeatures("tnum")
			})
		}
	})
}

// labelBars shows the plays of each language or genre as bars in the
// accent, the most first; a language its tracks don't tell is "other".
func (s *shell) labelBars(c *ui.Context, rows []library.LabelPlays) {
	k := &s.lk
	if len(rows) == 0 {
		ui.Text(c, "—").TextColor(k.muted)
		return
	}
	top := float32(rows[0].Plays)
	ui.Column(c).Gap(8).FillWidth().Children(func() {
		for _, r := range rows {
			label := cmp.Or(r.Label, locales.StatsOtherLanguage())
			ui.Row(c).Gap(12).AlignItems(ui.Center).Label(locales.StatsArtistPlays(label, r.Plays)).Children(func() {
				ui.Text(c, label).FontSize(k.smallSize()).Width(72).Shrink(0).SingleLine()
				frac := float32(r.Plays) / top
				ui.Box(c).Grow(1).Basis(0).Height(10).Draw(func(p *ui.Painter, rect ui.Rect) {
					p.Fill(ui.Rect{X: rect.X, Y: rect.Y, W: max(2, rect.W*frac), H: rect.H}, k.accent, 3)
				})
				ui.Text(c, strconv.Itoa(r.Plays)).FontSize(k.smallSize()).TextColor(k.muted).FontFeatures("tnum").Width(40).Shrink(0)
			})
		}
	})
}

// heatmapWeeks is how many weeks the heatmap shows: a year, and this week.
const heatmapWeeks = 53

// heatmap shows the plays of each day of the last year as GitHub shows
// contributions: a column of days for each week, Sunday first, the more
// plays the deeper in the accent; a day tells its minutes and plays as
// the pointer rests on it.
func (s *shell) heatmap(c *ui.Context) {
	k := &s.lk
	today := time.Now()
	days := s.svc.Collection.Daily((heatmapWeeks-1)*7 + int(today.Weekday()) + 1)
	most := 0
	for _, d := range days {
		most = max(most, d.Plays)
	}
	level := func(plays int) ui.Color {
		if plays == 0 {
			return k.surface3
		}
		return k.surface3.Mix(k.accent, 0.25+0.75*float32(plays)/float32(max(most, 1)))
	}
	const cell, gap, top = 11, 3, 16
	s.sectionTitle(c, locales.StatsHeatmap())
	ui.Text(c, locales.StatsHeatmapNote()).FontSize(k.smallSize()).TextColor(k.muted).Margin(0, 0, 12)
	ui.Column(c).Gap(4).Margin(0, 0, 48).Children(func() {
		// The weeks share the width, each day a square at most cell wide;
		// the month shows above the week its first day is in.
		grid := ui.Box(c).FillWidth().Height(top + 7*(cell+gap)).Draw(func(p *ui.Painter, r ui.Rect) {
			w := r.W / heatmapWeeks
			side := min(w, (r.H-top)/7) - gap
			for i, d := range days {
				x, y := r.X+float32(i/7)*w, r.Y+top+float32(i%7)*(r.H-top)/7
				p.Fill(ui.Rect{X: x, Y: y, W: side, H: side}, level(d.Plays), 2)
				if t, err := time.ParseInLocation(time.DateOnly, d.Date, time.Local); err == nil && t.Day() == 1 {
					p.Text(x, r.Y, monthName(t), k.captionSize(), k.muted)
				}
			}
		})
		// A box over each day, for the pointer to tell it.
		grid.Children(func() {
			ui.Row(c).Absolute().Top(top).Right(0).Bottom(0).Left(0).AlignItems(ui.Stretch).Children(func() {
				for w := 0; w < heatmapWeeks; w++ {
					ui.Column(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Stretch).Children(func() {
						for d := w * 7; d < min(w*7+7, len(days)); d++ {
							ui.Box(c).Grow(1).Basis(0).Tooltip(describeDay(days[d]))
						}
						for range w*7 + 7 - max(len(days), w*7) {
							ui.Box(c).Grow(1).Basis(0)
						}
					})
				}
			})
		})
		ui.Row(c).Gap(gap).AlignItems(ui.Center).Margin(4, 0, 0).Children(func() {
			ui.Text(c, locales.StatsLess()).FontSize(k.captionSize()).TextColor(k.muted).Margin(0, 4, 0, 0)
			for _, f := range []float32{0, 0.25, 0.5, 0.75, 1} {
				fill := k.surface3
				if f > 0 {
					fill = k.surface3.Mix(k.accent, 0.25+0.75*f)
				}
				ui.Box(c).Size(cell, cell).Radius(2).Background(fill)
			}
			ui.Text(c, locales.CollectionMore()).FontSize(k.captionSize()).TextColor(k.muted).Margin(0, 0, 0, 4)
		})
	})
}
