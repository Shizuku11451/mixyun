package audio

import (
	"io"
	"slices"

	"github.com/fzn0x/mixmusic/flac"
)

// flacCodec decodes the FLAC of a file kept on a computer.
type flacCodec struct {
	r   *flac.Reader
	pcm []float32
}

// openFLAC returns the track of a FLAC file. Its errors are worded for the
// player.
// It decodes at its own rate when that is one of native's.
func openFLAC(src io.ReadSeeker, native []int) (track, timing, error) {
	r, err := flac.NewReader(src)
	if err != nil {
		return nil, timing{}, readError(err)
	}
	i := r.Info()
	tm := timing{total: i.Duration()}
	if slices.Contains(native, i.SampleRate) {
		tm.rate = i.SampleRate
	}
	return newPCMTrack(&flacCodec{r: r}, i.SampleRate, i.Channels, tm)
}

func (c *flacCodec) seek(f int64) (int64, error) { return c.r.SeekSample(f) }

func (c *flacCodec) read() ([]float32, error) {
	fr, err := c.r.ReadFrame()
	if err != nil {
		return nil, err
	}
	ch, n := len(fr.Samples), len(fr.Samples[0])
	scale := 1 / float32(int64(1)<<(fr.BitsPerSample-1))
	c.pcm = grow(c.pcm, ch*n)
	for k, samples := range fr.Samples {
		for i, v := range samples {
			c.pcm[i*ch+k] = float32(v) * scale
		}
	}
	return c.pcm, nil
}
