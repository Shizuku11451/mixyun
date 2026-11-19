package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/audio"
	"github.com/fzn0x/mixmusic/locales"
)

// headphoneEQRows import and show a headphone profile, in the equalizer's
// group of Settings › Audio (#22).
func (s *shell) headphoneEQRows(c *ui.Context) {
	h := &s.set.HeadphoneEQ
	desc := locales.HeadphoneEQNone()
	if h.Profile != "" {
		desc = h.Name
	}
	s.row(c, locales.HeadphoneEQ(), locales.HeadphoneEQDesc(), func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if h.Profile != "" && s.switchOf(c.Key("peq-on"), &h.Enabled).Label(locales.HeadphoneEQ()).Changed() {
				s.save()
			}
			if s.button(c, "peq-import", locales.HeadphoneEQImport(), false).Label(locales.HeadphoneEQImportTitle()).Clicked() {
				s.importHeadphoneEQ()
			}
		})
	}, "autoeq", "parametric", "equalizer apo", "headphones", desc)
	if s.peqProblem != "" {
		s.found(c, "", s.peqProblem, func() {
			ui.Text(c, s.peqProblem+" "+locales.HeadphoneEQKept()).FontSize(s.lk.smallSize()).TextColor(s.lk.danger).Padding(0, 0, 12).MaxWidth(720)
		})
	}
	p, err := audio.ParsePEQ(h.Profile)
	if h.Profile == "" || err != nil {
		return
	}
	chain := audio.Chain{EQ: s.set.Equalizer.Enabled, EQPreamp: s.set.Equalizer.Preamp, Attenuation: h.Attenuation}
	copy(chain.Gains[:], s.set.Equalizer.Gains)
	if h.Enabled {
		chain.PEQ = &p
	}
	summary := locales.HeadphoneEQSummary(h.Name, signedDB(p.Preamp), fmt.Sprint(len(p.Filters)))
	bands := locales.HeadphoneEQAlone()
	if chain.EQ {
		bands = locales.HeadphoneEQWithBands()
	}
	peak, at := chain.Peak()
	hz := fmt.Sprintf("%.0f Hz", at)
	if at >= 1000 {
		hz = fmt.Sprintf("%.1f kHz", at/1000)
	}
	clips := peak > 0.05
	headroom := locales.HeadphoneEQHeadroom(signedDB(peak), hz)
	if clips {
		headroom = locales.HeadphoneEQClips(signedDB(peak), hz)
	}
	s.found(c, "", summary+" "+bands+" "+headroom, func() {
		k := &s.lk
		ui.Column(c).Gap(6).Padding(0, 0, 14).MaxWidth(720).Children(func() {
			ui.Text(c, summary+" "+bands).FontSize(k.smallSize()).TextColor(k.muted)
			if h.Enabled {
				s.responseCurve(c, chain)
				ink := k.muted
				if clips {
					ink = k.danger
				}
				ui.Text(c, headroom).FontSize(k.smallSize()).TextColor(ink)
			}
			ui.Row(c).Gap(16).Children(func() {
				if h.Enabled && clips {
					// Half a dB over the estimate: the curve is sampled.
					by := math.Ceil(peak*2) / 2
					if s.link(c, "peq-attenuate", locales.HeadphoneEQAttenuate(signedDB(-by))) {
						h.Attenuation = max(h.Attenuation-by, -24)
						s.save()
					}
				}
				if h.Attenuation < 0 {
					ui.Text(c, locales.HeadphoneEQAttenuation(signedDB(h.Attenuation))).FontSize(k.smallSize()).TextColor(k.muted)
					if s.link(c, "peq-unattenuate", locales.HeadphoneEQRemoveAttenuation()) {
						h.Attenuation = 0
						s.save()
					}
				}
			})
		})
	})
}

// signedDB is a gain as "+2.5 dB".
func signedDB(db float64) string { return fmt.Sprintf("%+.1f dB", db) }

// responseCurve draws the gain of chain across 20 Hz to 20 kHz, from -12
// to +12 dB, with 0 dB marked.
func (s *shell) responseCurve(c *ui.Context, chain audio.Chain) {
	t := &s.lk.theme
	const points = 160
	var db [points + 1]float64
	for i := range db {
		db[i] = chain.ResponseDB(20 * math.Pow(1000, float64(i)/points))
	}
	ui.Box(c).Height(96).MaxWidth(480).Label(locales.HeadphoneEQCurve()).Draw(func(p *ui.Painter, r ui.Rect) {
		y := func(v float64) float32 { return r.Y + r.H*float32(0.5-min(max(v, -12), 12)/24) }
		p.Fill(ui.Rect{X: r.X, Y: y(0), W: r.W, H: 1}, t.Border, 0)
		var path ui.Path
		for i, v := range db {
			x := r.X + r.W*float32(i)/points
			if i == 0 {
				path.MoveTo(x, y(v))
			} else {
				path.LineTo(x, y(v))
			}
		}
		p.StrokePath(&path, 2, t.Accent)
	})
}

// importHeadphoneEQ asks for a ParametricEQ.txt file beside the main
// thread, and plays it once it reads whole; one that doesn't leaves the
// profile as it was and tells why.
func (s *shell) importHeadphoneEQ() {
	win := s.win
	go func() {
		paths, err := openFileDialog(mygo.OpenDialogOptions{Parent: win, Title: locales.HeadphoneEQImportTitle(),
			Filters: []mygo.FileFilter{{Name: locales.HeadphoneEQTextFiles(), Extensions: []string{"txt"}}}})
		if err == nil && len(paths) == 0 {
			return
		}
		var text []byte
		if err == nil {
			text, err = readSmall(paths[0], 1<<20)
		}
		problem := ""
		if err != nil {
			problem = locales.HeadphoneEQErrRead(err.Error())
		} else if _, perr := audio.ParsePEQ(string(text)); perr != nil {
			problem = peqProblem(perr)
		}
		s.update(func() {
			s.peqProblem = problem
			if problem != "" {
				return
			}
			name := strings.TrimSuffix(filepath.Base(paths[0]), filepath.Ext(paths[0]))
			s.set.HeadphoneEQ.Enabled, s.set.HeadphoneEQ.Name, s.set.HeadphoneEQ.Profile = true, name, string(text)
			s.set.HeadphoneEQ.Attenuation = 0
			s.save()
		})
	}()
}

// readSmall reads a file of at most limit bytes.
func readSmall(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(b)) > limit {
		err = errors.New("over 1 MB")
	}
	return b, err
}

// peqProblem tells why a profile didn't import.
func peqProblem(err error) string {
	var pe *audio.PEQError
	if !errors.As(err, &pe) {
		return err.Error()
	}
	line := fmt.Sprint(pe.Line)
	switch pe.Problem {
	case audio.PEQBadType:
		return locales.HeadphoneEQErrType(line, pe.Text)
	case audio.PEQBadNumber:
		return locales.HeadphoneEQErrNumber(line, pe.Text)
	case audio.PEQTooMany:
		return locales.HeadphoneEQErrMany(line)
	case audio.PEQEmpty:
		return locales.HeadphoneEQErrEmpty()
	}
	return locales.HeadphoneEQErrUnsupported(line, pe.Text)
}
