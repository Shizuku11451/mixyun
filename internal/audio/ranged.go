package audio

import (
	"io"
	"time"
)

// Ranged is a Source that plays only a part of its file, from Start to End,
// as a track of a CUE sheet plays its part of the album's (#24): End 0 is
// the file's end. Its times, its seeks and its length are the part's.
type Ranged interface {
	Range() (start, end time.Duration)
}

// rangedTrack is the part of a track from off, a frame of its stream, to
// limit, none when 0: the frames it decodes are those of the part, and it
// ends where the part does, the frames past it dropped.
type rangedTrack struct {
	track
	off, limit int64
	hz         int64 // the rate of its frames
	// at is the frame of the stream of the next frame decoded, -1 until a
	// packet after a seek tells it; over is set once the part ended.
	at   int64
	over bool
}

// ranged has tr play from start to end, and its timing tell the part's.
func ranged(tr track, tm timing, start, end time.Duration) (track, timing) {
	hz := tm.hz()
	r := &rangedTrack{track: tr, off: framesAt(start, hz), hz: hz, at: -1}
	if end > start {
		r.limit = framesAt(end, hz) + framesAt(tm.delay, hz)
		tm.total = end - start + tm.delay
	} else if tm.total > start {
		tm.total -= start
	}
	return r, tm
}

func (r *rangedTrack) seek(t time.Duration) error {
	r.at, r.over = -1, false
	return r.track.seek(t + durationAt(r.off, r.hz))
}

func (r *rangedTrack) read() error {
	if r.over {
		return io.EOF
	}
	return r.track.read()
}

func (r *rangedTrack) decode() (decoded, error) {
	p, err := r.track.decode()
	if err != nil {
		return p, err
	}
	if r.at < 0 {
		r.at = p.at
	}
	n := len(p.pcm) / r.channels()
	p.from, p.to = r.clip(p.from, n), r.clip(p.to, n)
	p.at -= r.off
	r.at += int64(n)
	return p, nil
}

// clip is i, a frame of the n decoded next, or the end of the part when
// it comes first, which then is over.
func (r *rangedTrack) clip(i, n int) int {
	if r.limit == 0 || r.at+int64(n) < r.limit {
		return i
	}
	r.over = true
	return min(i, int(max(0, r.limit-r.at)))
}

func (r *rangedTrack) end() []float32 {
	if r.over {
		return nil
	}
	pcm := r.track.end()
	ch := r.channels()
	return pcm[:ch*r.clip(len(pcm)/ch, len(pcm)/ch)]
}
