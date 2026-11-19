package localfiles

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fzn0x/mixmusic/internal/media"
)

// A track of a CUE sheet is a part of one FLAC file beside it (#24). Its
// ID names the file and the part, in CD frames of 1/75 s, so that it plays
// wherever it's kept without the sheet: "file:cue:<path>:<start>:<end>",
// end 0 for the end of the file.
const cuePrefix = "cue:"

// CueTrackID returns the ID of the part of the file at path from start to
// end, in CD frames.
func CueTrackID(path string, start, end int64) string {
	return media.ID(media.SourceFile, fmt.Sprintf("%s%s:%d:%d", cuePrefix, b64.EncodeToString([]byte(filepath.Clean(path))), start, end))
}

// CuePartOf returns the file and the part of the track of a CUE sheet's ID,
// and false for another ID.
func CuePartOf(id string) (path string, start, end time.Duration, ok bool) {
	source, native, err := media.ParseID(id)
	if err != nil || source != media.SourceFile || !strings.HasPrefix(native, cuePrefix) {
		return "", 0, 0, false
	}
	f := strings.Split(strings.TrimPrefix(native, cuePrefix), ":")
	if len(f) != 3 {
		return "", 0, 0, false
	}
	b, err := b64.DecodeString(f[0])
	a, err1 := strconv.ParseInt(f[1], 10, 64)
	z, err2 := strconv.ParseInt(f[2], 10, 64)
	if err != nil || err1 != nil || err2 != nil || !filepath.IsAbs(string(b)) || a < 0 || z != 0 && z <= a {
		return "", 0, 0, false
	}
	return string(b), cdTime(a), cdTime(z), true
}

// cdTime is a time of n CD frames.
func cdTime(n int64) time.Duration { return time.Duration(n) * time.Second / 75 }

// IsCue reports whether a file's name is a CUE sheet's.
func IsCue(path string) bool { return strings.EqualFold(filepath.Ext(path), ".cue") }

// CueProblem is why a CUE sheet doesn't open.
type CueProblem string

const (
	CueUnsupported CueProblem = "unsupported" // more than one file, a data track, a gap to add
	CueBadFile     CueProblem = "file"        // a file not a FLAC beside the sheet
	CueMissing     CueProblem = "missing"     // its FLAC isn't there or doesn't play
	CueBadIndex    CueProblem = "index"       // a time it can't read, or out of order
	CueOutOfRange  CueProblem = "range"       // a track that starts past the end of the file
	CueNoTracks    CueProblem = "empty"       // no track with an INDEX 01
)

// CueError tells the line of a sheet that failed, from 1 (0 for the whole
// sheet), and why.
type CueError struct {
	Line    int
	Text    string
	Problem CueProblem
}

func (e *CueError) Error() string {
	return fmt.Sprintf("cue sheet line %d: %s: %q", e.Line, e.Problem, e.Text)
}

// ReadCue reads the CUE sheet at path, of one FLAC file beside it: its
// tracks, in order, each the part of the file from its INDEX 01 to the
// next one's, and named as the sheet names them. The files aren't changed.
func (x *Index) ReadCue(path string) ([]media.Track, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 1<<20))
	if err != nil {
		return nil, err
	}
	type cueTrack struct {
		number           int
		title, performer string
		start            int64
		line             int
	}
	var (
		file, title, performer string
		tracks                 []cueTrack
		cur                    *cueTrack
	)
	text := strings.TrimPrefix(string(b), "\xef\xbb\xbf")
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		fail := func(why CueProblem) ([]media.Track, error) {
			return nil, &CueError{Line: n + 1, Text: line, Problem: why}
		}
		w := cueWords(line)
		if len(w) == 0 {
			continue
		}
		arg := func(i int) string {
			if i < len(w) {
				return w[i]
			}
			return ""
		}
		switch strings.ToUpper(w[0]) {
		case "REM", "CATALOG", "CDTEXTFILE", "SONGWRITER", "FLAGS", "ISRC":
		case "FILE":
			if file != "" {
				return fail(CueUnsupported) // a sheet of more than one file
			}
			name := arg(1)
			if name == "" || filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) || !strings.EqualFold(filepath.Ext(name), ".flac") {
				return fail(CueBadFile)
			}
			file = filepath.Join(filepath.Dir(path), name)
		case "TITLE":
			if cur != nil {
				cur.title = arg(1)
			} else {
				title = arg(1)
			}
		case "PERFORMER":
			if cur != nil {
				cur.performer = arg(1)
			} else {
				performer = arg(1)
			}
		case "TRACK":
			num, err := strconv.Atoi(arg(1))
			if file == "" || err != nil || !strings.EqualFold(arg(2), "AUDIO") {
				return fail(CueUnsupported)
			}
			tracks = append(tracks, cueTrack{number: num, start: -1, line: n + 1})
			cur = &tracks[len(tracks)-1]
		case "INDEX":
			num, err := strconv.Atoi(arg(1))
			if cur == nil || err != nil {
				return fail(CueBadIndex)
			}
			if num != 1 {
				continue // INDEX 00 is the gap before it, which the track before keeps
			}
			at, ok := cueFrames(arg(2))
			if !ok || len(tracks) > 1 && at <= tracks[len(tracks)-2].start {
				return fail(CueBadIndex)
			}
			cur.start = at
		case "PREGAP", "POSTGAP":
			return fail(CueUnsupported) // silence that the file hasn't
		default:
			return fail(CueUnsupported)
		}
	}
	for _, t := range tracks {
		if t.start < 0 {
			return nil, &CueError{Line: t.line, Text: fmt.Sprintf("TRACK %02d", t.number), Problem: CueBadIndex}
		}
	}
	if len(tracks) == 0 {
		return nil, &CueError{Problem: CueNoTracks}
	}
	e, err := x.ReadFile(file)
	if err != nil {
		return nil, &CueError{Text: filepath.Base(file), Problem: CueMissing}
	}
	length := time.Duration(e.DurationMs) * time.Millisecond
	out := make([]media.Track, len(tracks))
	for i, ct := range tracks {
		if length > 0 && cdTime(ct.start) >= length {
			return nil, &CueError{Line: ct.line, Text: fmt.Sprintf("TRACK %02d", ct.number), Problem: CueOutOfRange}
		}
		end := int64(0)
		if i+1 < len(tracks) {
			end = tracks[i+1].start
		}
		t := e.track()
		t.ID = CueTrackID(file, ct.start, end)
		t.Title = ct.title
		if t.Title == "" {
			t.Title = fmt.Sprintf("Track %02d", ct.number)
		}
		if p := cmp.Or(ct.performer, performer); p != "" {
			t.Artists = []media.Ref{{Name: p}}
		}
		if title != "" && t.Album == nil {
			t.Album = &media.Ref{Name: title}
		}
		stop := length
		if end > 0 {
			stop = cdTime(end)
		}
		t.DurationMs = int(max(0, stop-cdTime(ct.start)) / time.Millisecond)
		out[i] = t
	}
	return out, nil
}

// cueFrames reads a time of a sheet, mm:ss:ff, in CD frames.
func cueFrames(s string) (int64, bool) {
	p := strings.Split(s, ":")
	if len(p) != 3 {
		return 0, false
	}
	var v [3]int64
	for i, f := range p {
		n, err := strconv.ParseInt(f, 10, 64)
		if err != nil || n < 0 {
			return 0, false
		}
		v[i] = n
	}
	if v[1] >= 60 || v[2] >= 75 {
		return 0, false
	}
	return (v[0]*60+v[1])*75 + v[2], true
}

// cueWords splits a line of a sheet into its words, a quoted one whole.
func cueWords(line string) []string {
	var w []string
	for line = strings.TrimSpace(line); line != ""; line = strings.TrimSpace(line) {
		if line[0] == '"' {
			end := strings.IndexByte(line[1:], '"')
			if end < 0 {
				w, line = append(w, line[1:]), ""
				continue
			}
			w, line = append(w, line[1:1+end]), line[2+end:]
			continue
		}
		word, rest, _ := strings.Cut(line, " ")
		w, line = append(w, word), rest
	}
	return w
}
