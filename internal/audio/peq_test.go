package audio

import (
	"errors"
	"math"
	"testing"
)

// autoEq is a profile as AutoEq writes ParametricEQ.txt.
const autoEq = `Preamp: -6.4 dB
Filter 1: ON LSC Fc 105 Hz Gain 5.8 dB Q 0.70
Filter 2: ON PK Fc 3000 Hz Gain -3.1 dB Q 1.41
Filter 3: OFF PK Fc 5000 Hz Gain 9 dB Q 2
Filter 4: ON HSC Fc 10000 Hz Gain -4.0 dB Q 0.70
`

func TestParsePEQ(t *testing.T) {
	p, err := ParsePEQ(autoEq)
	if err != nil {
		t.Fatal(err)
	}
	want := []PEQFilter{{"LSC", 105, 5.8, 0.7}, {"PK", 3000, -3.1, 1.41}, {"HSC", 10000, -4, 0.7}}
	if p.Preamp != -6.4 || len(p.Filters) != len(want) {
		t.Fatalf("got %+v", p)
	}
	for i, f := range want {
		if p.Filters[i] != f {
			t.Errorf("filter %d: %+v, want %+v", i, p.Filters[i], f)
		}
	}
}

// TestParsePEQFails names the line that can't run as it says, and why.
func TestParsePEQFails(t *testing.T) {
	for _, tc := range []struct {
		text string
		line int
		why  PEQProblem
	}{
		{"Preamp: -3 dB\nDevice: Speakers", 2, PEQUnsupported},
		{"Filter 1: ON BP Fc 100 Hz Gain 3 dB Q 1", 1, PEQBadType},
		{"Filter 1: ON LSC 12 dB Fc 100 Hz Gain 3 dB", 1, PEQBadNumber},
		{"Filter 1: ON PK Fc 100 Hz Gain 3 dB", 1, PEQBadNumber},
		{"Filter 1: ON PK Fc 100 Hz Gain 45 dB Q 1", 1, PEQBadNumber},
		{"Filter 1: ON PK Fc 30000 Hz Gain 3 dB Q 1", 1, PEQBadNumber},
		{"Filter 1: ON PK Fc 100 Hz Gain 3 dB Q NaN", 1, PEQBadNumber},
		{"Preamp: lots", 1, PEQBadNumber},
		{"# nothing\n\n", 0, PEQEmpty},
		{"GraphicEQ: 20 -1; 40 2", 1, PEQUnsupported},
	} {
		_, err := ParsePEQ(tc.text)
		var pe *PEQError
		if !errors.As(err, &pe) || pe.Line != tc.line || pe.Problem != tc.why {
			t.Errorf("%q: %v, want line %d %s", tc.text, err, tc.line, tc.why)
		}
	}
	many := ""
	for range maxPEQFilters + 1 {
		many += "Filter: ON PK Fc 100 Hz Gain 1 dB Q 1\n"
	}
	if _, err := ParsePEQ(many); err == nil {
		t.Error("too many filters imported")
	}
}

// TestPEQResponse checks each filter's gain where it is defined.
func TestPEQResponse(t *testing.T) {
	near := func(what string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.05 {
			t.Errorf("%s: %.3f dB, want %.3f", what, got, want)
		}
	}
	pk, ls, hs := PEQFilter{"PK", 1000, 6, 2}.filter(), PEQFilter{"LSC", 200, -8, 0.7}.filter(), PEQFilter{"HSC", 4000, 5, 0.7}.filter()
	near("peak at its frequency", pk.response(1000), 6)
	near("peak far off", pk.response(50), 0)
	near("low shelf below", ls.response(20), -8)
	near("low shelf at its frequency", ls.response(200), -4)
	near("high shelf above", hs.response(20000), 5)
	near("high shelf below", hs.response(100), 0)
}

// TestPEQPeakMeasured runs a sine near full scale through stacked boosts:
// what the stage plays peaks where Chain.Peak said, and the attenuation
// it suggests keeps it under full scale.
func TestPEQPeakMeasured(t *testing.T) {
	p, err := ParsePEQ("Preamp: -2 dB\nFilter 1: ON PK Fc 1000 Hz Gain 4 dB Q 1\nFilter 2: ON PK Fc 1000 Hz Gain 3 dB Q 1")
	if err != nil {
		t.Fatal(err)
	}
	c := Chain{EQ: true, PEQ: &p}
	c.Gains[5] = 2 // the ten bands' 1 kHz stacks on the profile's
	db, hz := c.Peak()
	if db < 6.5 || hz < 800 || hz > 1250 {
		t.Fatalf("peak %.2f dB at %.0f Hz", db, hz)
	}
	measure := func(c Chain) float64 {
		eq, s := newEqualizer(), newPEQStage(*c.PEQ, c.Attenuation)
		eq.set(c.EQ, c.Gains, c.EQPreamp)
		const amp = 0.99
		top := 0.0
		for q := range 400 {
			buf := make([]float32, 2*quantum)
			for i := range quantum {
				v := float32(amp * math.Sin(2*math.Pi*hz*float64(q*quantum+i)/Rate))
				buf[2*i], buf[2*i+1] = v, v
			}
			eq.process(buf, 2)
			s.process(buf, 2)
			if q > 200 {
				for _, v := range buf {
					top = max(top, math.Abs(float64(v)))
				}
			}
		}
		return 20 * math.Log10(top/amp)
	}
	if got := measure(c); math.Abs(got-db) > 0.1 {
		t.Errorf("measured %.2f dB, estimated %.2f", got, db)
	}
	c.Attenuation = -math.Ceil(db*2) / 2
	if got := measure(c); got > 0.05+20*math.Log10(1/0.99) {
		t.Errorf("attenuated by %.1f dB, it still peaks at %.2f dB", -c.Attenuation, got)
	}
	off := c
	off.PEQ = nil
	if got := off.ResponseDB(hz); math.Abs(got-2) > 0.1 {
		t.Errorf("bypassed, the chain is %.2f dB at %.0f Hz, the ten bands' alone", got, hz)
	}
}
