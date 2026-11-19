package audio

import (
	"fmt"
	"math"
	"math/cmplx"
	"strconv"
	"strings"
)

// PEQ is a headphone correction, as Equalizer APO's ParametricEQ.txt, which
// AutoEq writes, tells it: a preamp and peaking and shelving filters (#22).
type PEQ struct {
	Preamp  float64
	Filters []PEQFilter
}

// PEQFilter is one filter of a PEQ: Kind is "PK", "LSC" or "HSC".
type PEQFilter struct {
	Kind        string
	Hz, Gain, Q float64
}

// PEQProblem is why a line of a profile can't be imported.
type PEQProblem string

const (
	PEQUnsupported PEQProblem = "unsupported" // a line of APO's this player doesn't run
	PEQBadType     PEQProblem = "type"        // a filter other than PK, LSC or HSC
	PEQBadNumber   PEQProblem = "number"      // a number missing, unreadable or out of its range
	PEQTooMany     PEQProblem = "many"        // more filters than maxPEQFilters
	PEQEmpty       PEQProblem = "empty"       // no preamp nor filter
)

// PEQError tells the line of a profile that failed, from 1, and why.
type PEQError struct {
	Line    int
	Text    string
	Problem PEQProblem
}

func (e *PEQError) Error() string {
	return fmt.Sprintf("line %d: %s: %q", e.Line, e.Problem, e.Text)
}

const maxPEQFilters = 64

// ParsePEQ reads a ParametricEQ.txt profile. A line it can't run exactly,
// as a filter of another kind, a slope in place of Q or another of APO's
// commands, fails the whole profile rather than play another curve.
func ParsePEQ(text string) (PEQ, error) {
	var p PEQ
	any := false
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		fail := func(why PEQProblem) (PEQ, error) { return PEQ{}, &PEQError{Line: n + 1, Text: line, Problem: why} }
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rest, ok := strings.Cut(line, ":")
		key = strings.ToLower(strings.TrimSpace(key))
		f := strings.Fields(rest)
		switch {
		case ok && key == "preamp":
			if len(f) != 2 || !strings.EqualFold(f[1], "db") {
				return fail(PEQBadNumber)
			}
			v, ok := number(f[0], -60, 30)
			if !ok {
				return fail(PEQBadNumber)
			}
			p.Preamp += v // APO adds up its preamps
			any = true
		case ok && (key == "filter" || strings.HasPrefix(key, "filter ")):
			if len(f) < 2 {
				return fail(PEQBadNumber)
			}
			switch strings.ToUpper(f[0]) {
			case "OFF":
				any = true
				continue
			case "ON":
			default:
				return fail(PEQUnsupported)
			}
			kind := strings.ToUpper(f[1])
			if kind != "PK" && kind != "LSC" && kind != "HSC" {
				return fail(PEQBadType)
			}
			// Exactly Fc <hz> Hz Gain <db> dB Q <q>: a slope in dB, as
			// "LSC 12 dB", or a bandwidth isn't a Q.
			if len(f) != 10 || !strings.EqualFold(f[2], "fc") || !strings.EqualFold(f[4], "hz") ||
				!strings.EqualFold(f[5], "gain") || !strings.EqualFold(f[7], "db") || !strings.EqualFold(f[8], "q") {
				return fail(PEQBadNumber)
			}
			hz, ok1 := number(f[3], 10, 20000)
			gain, ok2 := number(f[6], -30, 30)
			q, ok3 := number(f[9], 0.05, 30)
			if !ok1 || !ok2 || !ok3 {
				return fail(PEQBadNumber)
			}
			if len(p.Filters) == maxPEQFilters {
				return fail(PEQTooMany)
			}
			p.Filters = append(p.Filters, PEQFilter{Kind: kind, Hz: hz, Gain: gain, Q: q})
			any = true
		default:
			return fail(PEQUnsupported)
		}
	}
	if !any {
		return PEQ{}, &PEQError{Problem: PEQEmpty}
	}
	return p, nil
}

// number reads a finite number from lo to hi.
func number(s string, lo, hi float64) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil && v >= lo && v <= hi
}

// filter is f as a biquad: the RBJ cookbook's peaking and shelving filters
// of quality Q, as APO runs them.
func (f PEQFilter) filter() biquad {
	var b biquad
	a := math.Pow(10, f.Gain/40)
	w0 := 2 * math.Pi * f.Hz / Rate
	alpha, k := math.Sin(w0)/(2*f.Q), math.Cos(w0)
	k2 := 2 * math.Sqrt(a) * alpha
	ap1, am1 := a+1, a-1
	switch f.Kind {
	case "LSC":
		b.normalize(a*(ap1-am1*k+k2), 2*a*(am1-ap1*k), a*(ap1-am1*k-k2), ap1+am1*k+k2, -2*(am1+ap1*k), ap1+am1*k-k2)
	case "HSC":
		b.normalize(a*(ap1+am1*k+k2), -2*a*(am1+ap1*k), a*(ap1+am1*k-k2), ap1-am1*k+k2, 2*(am1-ap1*k), ap1-am1*k-k2)
	default:
		b.normalize(1+alpha*a, -2*k, 1-alpha*a, 1+alpha/a, -2*k, 1-alpha/a)
	}
	return b
}

// response is the gain of b at hz, in dB.
func (b *biquad) response(hz float64) float64 {
	z := cmplx.Exp(complex(0, -2*math.Pi*hz/Rate))
	h := (complex(b.b0, 0) + complex(b.b1, 0)*z + complex(b.b2, 0)*z*z) / (1 + complex(b.a1, 0)*z + complex(b.a2, 0)*z*z)
	return 20 * math.Log10(cmplx.Abs(h))
}

// peqStage runs a PEQ: its preamp and an attenuation of the user's, then
// its filters.
type peqStage struct {
	gain    float32
	filters []biquad
}

func newPEQStage(p PEQ, attenuation float64) *peqStage {
	s := &peqStage{gain: float32(math.Pow(10, (p.Preamp+attenuation)/20))}
	for _, f := range p.Filters {
		s.filters = append(s.filters, f.filter())
	}
	return s
}

func (s *peqStage) process(buf []float32, ch int) {
	if s.gain != 1 {
		for i := range buf {
			buf[i] *= s.gain
		}
	}
	for i := range s.filters {
		s.filters[i].process(buf, ch)
	}
}

func (s *peqStage) forget(from int) {
	for i := range s.filters {
		s.filters[i].forget(from)
	}
}

// Chain is what shapes the sound before the volume, for its curve: the
// equalizer's ten bands when EQ, and a headphone profile when PEQ isn't
// nil, with an attenuation in dB.
type Chain struct {
	EQ          bool
	Gains       [len(Bands)]float64
	EQPreamp    float64
	PEQ         *PEQ
	Attenuation float64
}

// ResponseDB is the gain of the chain at hz, in dB.
func (c Chain) ResponseDB(hz float64) float64 {
	db := 0.0
	if c.EQ {
		db += c.EQPreamp
		for i, g := range c.Gains {
			var b biquad
			b.set(bandKind(i), Bands[i], bandQ, g)
			db += b.response(hz)
		}
	}
	if c.PEQ != nil {
		db += c.PEQ.Preamp + c.Attenuation
		for _, f := range c.PEQ.Filters {
			b := f.filter()
			db += b.response(hz)
		}
	}
	return db
}

// Peak is the most the chain turns a sine up, in dB, and at which
// frequency, from 20 Hz to 20 kHz: an estimate of the headroom a track at
// full scale lacks, not a true peak.
func (c Chain) Peak() (db, hz float64) {
	db = math.Inf(-1)
	for i := range 481 {
		f := 20 * math.Pow(1000, float64(i)/480)
		if r := c.ResponseDB(f); r > db {
			db, hz = r, f
		}
	}
	return db, hz
}
