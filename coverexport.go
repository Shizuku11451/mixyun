package main

// Export cover…, in the menu of a track, asks where to save its cover at
// the largest size its service gives, and writes it there as a PNG.

import (
	"context"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/locales"
)

// exportCoverItem is Export cover… in the menu of a track with a cover.
func (s *shell) exportCoverItem(m *ui.Menu, t media.Track) {
	if len(t.Artwork) > 0 && m.Item(locales.PanelExportCover()).Chosen() {
		s.exportCover(t)
	}
}

// coverURL is the address of the largest image of a cover: NetEase's
// own, without the size its addresses ask for.
func coverURL(t media.Track) string {
	u := artworkURL(t.Artwork, math.MaxInt32)
	if t.Source == media.SourceNetEase {
		u, _, _ = strings.Cut(u, "?param=")
	}
	return u
}

func (s *shell) exportCover(t media.Track) {
	win, u := s.win, coverURL(t)
	name := cardFileName(t.ArtistNames() + " - " + t.Title)
	go func() {
		path, err := saveFileDialog(mygo.SaveDialogOptions{Parent: win, Title: locales.ShareSaveTitle(), DefaultPath: name,
			ButtonLabel: locales.ShareSaveButton(), Filters: []mygo.FileFilter{{Name: locales.SharePNGImage(), Extensions: []string{"png"}}}, CreateDirectories: true})
		if err == nil && path == "" {
			return
		}
		if err == nil {
			path, err = s.writeCover(path, u)
		}
		s.update(func() {
			if err != nil {
				s.notes = append(s.notes, noticeOf(err))
				return
			}
			s.notes = append(s.notes, notice(locales.ShareSaved(filepath.Base(path))))
		})
	}()
}

// writeCover writes the image at u to path, with .png at its end, and
// returns where it wrote.
func (s *shell) writeCover(path, u string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".png") {
		path += ".png"
	}
	ctx, cancel := context.WithTimeout(context.Background(), cardCoverWait)
	defer cancel()
	img, err := s.art.src.Image(ctx, u)
	if err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return "", err
	}
	return path, f.Close()
}
