package library

import (
	"slices"
	"testing"
	"time"

	"github.com/fzn0x/mixmusic/internal/media"
)

func TestStats(t *testing.T) {
	l := open(t)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }
	a, b := tr("yt:a", media.SourceYouTube, "A"), tr("sp:b", media.SourceSpotify, "B")
	l.RecordPlay(a)
	l.RecordPlay(b)
	l.RecordPlay(a)
	s := l.Stats(0)
	if s.Plays != 3 || s.YouTube != 2 || s.Spotify != 1 || s.Minutes != 9 {
		t.Errorf("stats = %+v", s)
	}
	if s.TopTracks[0].Track.ID != "yt:a" || s.TopTracks[0].Plays != 2 || s.TopArtists[0].Artist.Name != "A" {
		t.Errorf("top = %+v %+v", s.TopTracks, s.TopArtists)
	}
	now = now.AddDate(0, 0, 10)
	if s := l.Stats(7); s.Plays != 0 {
		t.Errorf("plays older than a week counted: %+v", s)
	}
}

func TestStatsDaily(t *testing.T) {
	l := open(t)
	zone := time.FixedZone("UTC+7", 7*60*60)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, zone)
	l.now = func() time.Time { return now }
	a := tr("yt:a", media.SourceYouTube, "A")
	// Two plays on the 16th, local time; one just after midnight of the 18th.
	now = time.Date(2026, 10, 16, 9, 0, 0, 0, zone)
	l.RecordPlay(a)
	l.RecordPlay(a)
	now = time.Date(2026, 10, 18, 0, 30, 0, 0, zone)
	l.RecordPlay(a)
	now = time.Date(2026, 10, 18, 12, 0, 0, 0, zone)

	s := l.Stats(7)
	if len(s.Daily) != 7 || s.Daily[0].Date != "2026-10-12" || s.Daily[6].Date != "2026-10-18" {
		t.Fatalf("daily days = %+v", s.Daily)
	}
	if d := s.Daily[4]; d.Date != "2026-10-16" || d.Plays != 2 || d.Minutes != 6 {
		t.Errorf("the 16th = %+v", d)
	}
	if d := s.Daily[6]; d.Plays != 1 || d.Minutes != 3 {
		t.Errorf("the 18th = %+v", d)
	}
	if d := s.Daily[5]; d.Plays != 0 || d.Minutes != 0 {
		t.Errorf("a day without plays = %+v", d)
	}
	if all := l.Stats(0); len(all.Daily) != 90 || all.Daily[89].Plays != 1 {
		t.Errorf("all time covers %d days, last %+v", len(all.Daily), all.Daily[len(all.Daily)-1])
	}
}

func TestStatsNetEaseLanguagesGenres(t *testing.T) {
	l := open(t)
	jp := tr("ne:1", media.SourceNetEase, "A")
	jp.Title = "夜に駆ける"
	rock := tr("ne:2", media.SourceNetEase, "周杰伦")
	rock.Genre, rock.Language = "摇滚", "粤语"
	en := tr("sp:3", media.SourceSpotify, "Queen")
	for _, x := range []media.Track{jp, jp, rock, en} {
		l.RecordPlay(x)
	}
	s := l.Stats(0)
	if s.NetEase != 3 || s.Spotify != 1 {
		t.Errorf("netease %d spotify %d", s.NetEase, s.Spotify)
	}
	want := []LabelPlays{{"日语", 2}, {"粤语", 1}, {"英语", 1}}
	if !slices.Equal(s.Languages, want) {
		t.Errorf("languages = %v", s.Languages)
	}
	if !slices.Equal(s.Genres, []LabelPlays{{"摇滚", 1}}) {
		t.Errorf("genres = %v", s.Genres)
	}
	if d := l.Daily(400); len(d) != 400 || d[399].Plays != 4 {
		t.Errorf("daily = %d days, today %+v", len(d), d[len(d)-1])
	}
}
