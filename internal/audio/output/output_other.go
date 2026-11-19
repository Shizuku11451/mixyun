//go:build !((windows || darwin || linux) && (amd64 || arm64))

package output

import "time"

// Stream plays audio; there are none on this system.
type Stream struct{}

// Available returns ErrUnsupported.
func Available() error { return ErrUnsupported }

// Open returns ErrRate for a rate other than Rate, else ErrUnsupported.
func Open(o Options) (*Stream, error) {
	if o.Rate != 0 && o.Rate != Rate {
		return nil, ErrRate
	}
	return nil, ErrUnsupported
}

// Devices returns ErrUnsupported.
func Devices() ([]Device, error) { return nil, ErrUnsupported }

// Stop does nothing.
func (s *Stream) Stop() {}

// Delay returns 0.
func (s *Stream) Delay() time.Duration { return 0 }
