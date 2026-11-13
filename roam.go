package main

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/media"
	"github.com/fzn0x/mixmusic/internal/netease"
	"github.com/fzn0x/mixmusic/locales"
)

// roamPage shows NetEase's tags by era, language and genre; a tag plays
// its songs, shuffled. Signed in, a tile before them roams after what
// the user liked.
func (s *shell) roamPage(c *ui.Context) {
	s.pageScroll(c, func() {
		s.pageHeader(c, locales.RoamTitle(), nil, nil)
		s.roamTags(c)
	})
}

func (s *shell) roamTags(c *ui.Context) {
	titles := [3]string{locales.RoamByEra(), locales.RoamByLanguage(), locales.RoamByGenre()}
	n := 0
	if s.signedIn(media.SourceNetEase) {
		s.sectionTitle(c, locales.RoamForYou())
		ui.Grid(c.Key("roam-you")).Columns(s.moodColumns(c)).GapX(8).GapY(8).Margin(0, 0, 48).Children(func() {
			if name := locales.RoamLiked(); s.moodTile(c, "roam-liked", 3, name, locales.RoamStarting(name)) {
				s.roam(netease.RoamLiked, name)
			}
		})
	}
	for gi, tags := range netease.RoamTags {
		s.sectionTitle(c, titles[gi])
		ui.Grid(c.Key("roam-"+count(gi))).Columns(s.moodColumns(c)).GapX(8).GapY(8).Margin(0, 0, 48).Children(func() {
			for ti, tag := range tags {
				tone := n % 4
				n++
				if s.moodTile(c, "roam-"+count(gi)+"-"+count(ti), tone, tag, locales.RoamStarting(tag)) {
					s.roam(tag, tag)
				}
			}
		})
	}
}

// roam plays the songs of a tag, shuffled; name is the tag as its tile
// names it.
func (s *shell) roam(tag, name string) {
	s.notes = append(s.notes, notice(locales.RoamStarting(name)))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		tracks, err := s.svc.Catalog.Roam(ctx, tag)
		s.update(func() {
			switch {
			case err != nil:
				s.notes = append(s.notes, noticeOf(err))
			case len(tracks) == 0:
				s.notes = append(s.notes, notice(locales.RoamNothing(name)))
			default:
				// More of the tag follow where the queue runs out.
				s.p.PlayEndless(tracks, func(ctx context.Context) ([]media.Track, error) { return s.svc.Catalog.Roam(ctx, tag) })
			}
		})
	}()
}
