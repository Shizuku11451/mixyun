package library

import (
	"cmp"
	"slices"
	"time"
	"unicode"

	"github.com/fzn0x/mixmusic/internal/media"
)

const (
	// topCount is how many tracks and artists the stats rank.
	topCount = 10
	// dailyAllTime is how many days the daily plays cover for the whole
	// history.
	dailyAllTime = 90
)

// TrackPlays is a track and how often it was played.
type TrackPlays struct {
	Track media.Track `json:"track"`
	Plays int         `json:"plays"`
}

// ArtistPlays is an artist and how often their tracks were played.
type ArtistPlays struct {
	Artist media.Ref `json:"artist"`
	Plays  int       `json:"plays"`
}

// DayPlays is what the user played on one day of their own time zone.
type DayPlays struct {
	// Date is the day, as 2026-10-19.
	Date    string `json:"date"`
	Plays   int    `json:"plays"`
	Minutes int    `json:"minutes"`
}

// Stats sums up what the user listened to.
type Stats struct {
	Plays   int `json:"plays"`
	Minutes int `json:"minutes"`
	// YouTube and Spotify count the plays of each service's tracks.
	YouTube    int           `json:"youtube"`
	Spotify    int           `json:"spotify"`
	NetEase    int           `json:"netease"`
	TopTracks  []TrackPlays  `json:"topTracks"`
	TopArtists []ArtistPlays `json:"topArtists"`
	// Days is the number of days counted; 0 counts the whole history.
	Days int `json:"days"`
	// Daily holds every day of the period, the oldest first, those
	// without plays too: the last Days days, or the last 90 for the whole
	// history.
	Daily []DayPlays `json:"daily"`
	// Languages and Genres count the plays by language and by genre, the
	// most first. A track's language is its own (media.Track.Language),
	// else told by the letters of its title and artists, "" when they
	// tell none; tracks without a genre aren't in Genres.
	Languages []LabelPlays `json:"languages"`
	Genres    []LabelPlays `json:"genres"`
}

// LabelPlays is a language or a genre and how often its tracks played.
type LabelPlays struct {
	Label string `json:"label"`
	Plays int    `json:"plays"`
}

// Stats sums up the plays of the last days, or of the whole history when
// days is 0.
func (l *Library) Stats(days int) Stats {
	s := Stats{TopTracks: []TrackPlays{}, TopArtists: []ArtistPlays{}, Days: days}
	since := time.Time{}
	if days > 0 {
		since = l.now().AddDate(0, 0, -days)
	}
	tracks := map[string]*TrackPlays{}
	artists := map[string]*ArtistPlays{}
	languages, genres := map[string]int{}, map[string]int{}
	daily, dayIndex := dailyDays(l.now(), cmp.Or(days, dailyAllTime))
	dayMs := make([]int, len(daily))
	ms := 0
	for _, p := range l.file.Get().History {
		if p.PlayedAt.Before(since) {
			break // the history is newest first
		}
		if p.Track.IsEpisode() {
			continue // the stats are of music; the history keeps episodes
		}
		s.Plays++
		ms += p.Track.DurationMs
		if i, ok := dayIndex[p.PlayedAt.In(l.now().Location()).Format(time.DateOnly)]; ok {
			daily[i].Plays++
			dayMs[i] += p.Track.DurationMs
		}
		switch p.Track.Source {
		case media.SourceYouTube:
			s.YouTube++
		case media.SourceSpotify:
			s.Spotify++
		case media.SourceNetEase:
			s.NetEase++
		}
		languages[cmp.Or(p.Track.Language, languageOf(p.Track))]++
		if p.Track.Genre != "" {
			genres[p.Track.Genre]++
		}
		if tp := tracks[p.Track.ID]; tp != nil {
			tp.Plays++
		} else {
			tracks[p.Track.ID] = &TrackPlays{Track: p.Track, Plays: 1}
		}
		for _, a := range p.Track.Artists {
			// Services spell the same artist alike, not with the same ID.
			if ap := artists[a.Name]; ap != nil {
				ap.Plays++
			} else {
				artists[a.Name] = &ArtistPlays{Artist: a, Plays: 1}
			}
		}
	}
	s.Minutes = ms / 60000
	for i := range daily {
		daily[i].Minutes = dayMs[i] / 60000
	}
	s.Daily = daily
	for _, tp := range tracks {
		s.TopTracks = append(s.TopTracks, *tp)
	}
	for _, ap := range artists {
		s.TopArtists = append(s.TopArtists, *ap)
	}
	slices.SortFunc(s.TopTracks, func(a, b TrackPlays) int {
		return cmp.Or(b.Plays-a.Plays, cmp.Compare(a.Track.Title, b.Track.Title))
	})
	slices.SortFunc(s.TopArtists, func(a, b ArtistPlays) int {
		return cmp.Or(b.Plays-a.Plays, cmp.Compare(a.Artist.Name, b.Artist.Name))
	})
	s.Languages, s.Genres = ranked(languages), ranked(genres)
	s.TopTracks = s.TopTracks[:min(len(s.TopTracks), topCount)]
	s.TopArtists = s.TopArtists[:min(len(s.TopArtists), topCount)]
	return s
}

// dailyDays returns the last n days up to now's, in now's time zone, the
// oldest first, and the index of each by its date.
func dailyDays(now time.Time, n int) ([]DayPlays, map[string]int) {
	days := make([]DayPlays, n)
	index := make(map[string]int, n)
	y, m, d := now.Date()
	for i := range n {
		// Adding days to the calendar date stays right across clock changes.
		date := time.Date(y, m, d-(n-1-i), 12, 0, 0, 0, now.Location()).Format(time.DateOnly)
		days[i] = DayPlays{Date: date}
		index[date] = i
	}
	return days, index
}

// ranked lists counts the most first.
func ranked(counts map[string]int) []LabelPlays {
	out := make([]LabelPlays, 0, len(counts))
	for label, n := range counts {
		out = append(out, LabelPlays{label, n})
	}
	slices.SortFunc(out, func(a, b LabelPlays) int { return cmp.Or(b.Plays-a.Plays, cmp.Compare(a.Label, b.Label)) })
	return out
}

// languageOf tells a track's language by the letters of its title and
// artists, as NetEase's tags name it: kana are Japanese, hangul Korean,
// other Chinese characters Chinese, and Latin letters English; "" for
// none.
func languageOf(t media.Track) string {
	var han, latin bool
	for _, r := range t.Title + t.ArtistNames() {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			return "日语"
		case unicode.Is(unicode.Hangul, r):
			return "韩语"
		case unicode.Is(unicode.Han, r):
			han = true
		case unicode.Is(unicode.Latin, r):
			latin = true
		}
	}
	switch {
	case han:
		return "华语"
	case latin:
		return "英语"
	}
	return ""
}

// Daily returns what the user played on each of the last n days, the
// oldest first, those without plays too, as far as the history goes.
func (l *Library) Daily(n int) []DayPlays {
	days, index := dailyDays(l.now(), n)
	ms := make([]int, n)
	for _, p := range l.file.Get().History {
		if p.Track.IsEpisode() {
			continue
		}
		if i, ok := index[p.PlayedAt.In(l.now().Location()).Format(time.DateOnly)]; ok {
			days[i].Plays++
			ms[i] += p.Track.DurationMs
		}
	}
	for i := range days {
		days[i].Minutes = ms[i] / 60000
	}
	return days
}
