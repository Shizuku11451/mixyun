package lyrics

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	yrcLineRe = regexp.MustCompile(`^\[(\d+),\d+\]`)
	yrcWordRe = regexp.MustCompile(`\((\d+),(\d+),-?\d+\)`)
)

// ParseYRC reads lyrics in NetEase Cloud Music's YRC format, synchronized
// word by word: lines of "[start,length](start,length,0)word(…)word",
// their times in milliseconds. The credits before them, lines of JSON,
// are left out, as are lines without words.
func ParseYRC(yrc string) []Line {
	lines := []Line{}
	for raw := range strings.SplitSeq(strings.ReplaceAll(yrc, "\r\n", "\n"), "\n") {
		head := yrcLineRe.FindStringSubmatch(raw)
		if head == nil {
			continue
		}
		line := Line{}
		line.TimeMs, _ = strconv.Atoi(head[1])
		body := raw[len(head[0]):]
		tags := yrcWordRe.FindAllStringSubmatchIndex(body, -1)
		var text strings.Builder
		for i, tag := range tags {
			end := len(body)
			if i+1 < len(tags) {
				end = tags[i+1][0]
			}
			sung := body[tag[1]:end]
			text.WriteString(sung)
			// A word's text is as the line's holds it: without the space
			// after it, which lights with it.
			if sung = strings.TrimSpace(sung); sung != "" {
				start, _ := strconv.Atoi(body[tag[2]:tag[3]])
				length, _ := strconv.Atoi(body[tag[4]:tag[5]])
				line.Words = append(line.Words, Word{Text: sung, StartMs: start, EndMs: start + length})
			}
		}
		if line.Text = strings.TrimSpace(text.String()); line.Text != "" {
			lines = append(lines, line)
		}
	}
	slices.SortStableFunc(lines, func(a, b Line) int { return a.TimeMs - b.TimeMs })
	return lines
}
