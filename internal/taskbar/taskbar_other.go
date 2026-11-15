//go:build !(windows && (amd64 || arm64))

package taskbar

// Other systems have no buttons on their taskbar's thumbnails, nor 32-bit
// Windows here.
type bar struct{}

func attach(uintptr, func(Button)) (*bar, error) { return nil, ErrUnsupported }

func (*bar) show(state) {}

type lyricsWin struct{}

func openLyrics(int, func()) (*lyricsWin, error) { return nil, ErrUnsupported }
func (*lyricsWin) show(LyricsView)               {}
func (*lyricsWin) close()                        {}
func (*lyricsWin) isClosed() bool                { return true }
