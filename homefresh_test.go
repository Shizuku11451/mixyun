package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestFreshen asks again once the answer is older than the age, and the
// last ask older than freshRetry; never before the first answer, nor while
// an ask is out.
func TestFreshen(t *testing.T) {
	now := time.Date(2026, 11, 20, 7, 0, 0, 0, time.Local)
	for _, c := range []struct {
		name    string
		l       loaded[int]
		asks    bool
		waitFor time.Duration
	}{
		{"fresh", loaded[int]{done: true, at: now.Add(-20 * time.Minute), askedAt: now.Add(-20 * time.Minute)}, false, 40 * time.Minute},
		{"old", loaded[int]{done: true, at: now.Add(-2 * time.Hour), askedAt: now.Add(-2 * time.Hour)}, true, freshRetry},
		{"old, asked just now", loaded[int]{done: true, at: now.Add(-2 * time.Hour), askedAt: now.Add(-time.Minute)}, false, 4 * time.Minute},
		{"failed first", loaded[int]{done: true, err: errors.New("offline"), askedAt: now.Add(-6 * time.Minute)}, true, freshRetry},
		{"loading", loaded[int]{askedAt: now.Add(-2 * time.Hour)}, false, freshRetry},
		{"never asked", loaded[int]{done: true}, false, freshRetry},
		{"refreshing", loaded[int]{done: true, refreshing: true, at: now.Add(-2 * time.Hour), askedAt: now.Add(-2 * time.Hour)}, false, freshRetry},
	} {
		wait := c.l.freshen(time.Hour, now)
		if c.l.again != c.asks || (c.asks && !c.l.quiet) || wait != c.waitFor {
			t.Errorf("%s: asks %v, quiet %v, waits %v; want asks %v, waits %v", c.name, c.l.again, c.l.quiet, wait, c.asks, c.waitFor)
		}
	}
}

// TestKeepFresh asks again for what grew old: one that fails keeps what
// showed, without a toast, and isn't asked again before freshRetry; the
// next that comes shows.
func TestKeepFresh(t *testing.T) {
	s := newTestShell(t)
	answer, fail := "yesterday", false
	get := func(context.Context) (string, error) {
		if fail {
			return "", errors.New("offline")
		}
		return answer, nil
	}
	answered := func(l *loaded[string], asks int) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); l.answers < asks; time.Sleep(5 * time.Millisecond) {
			s.drain()
			if time.Now().After(deadline) {
				t.Fatalf("no answer to ask %d", asks)
			}
		}
	}
	l := fetch(s, "daily", get)
	answered(l, 1)
	defer func() {
		if s.freshTimer != nil {
			s.freshTimer.Stop()
		}
	}()
	age := func(d time.Duration) { l.at, l.askedAt = l.at.Add(-d), l.askedAt.Add(-d) }

	age(2 * time.Hour)
	fail = true
	s.keepFresh("daily", time.Hour)
	fetch(s, "daily", get)
	for deadline := time.Now().Add(5 * time.Second); l.quiet; time.Sleep(5 * time.Millisecond) {
		s.drain()
		if time.Now().After(deadline) {
			t.Fatal("the quiet ask didn't end")
		}
	}
	if l.value != "yesterday" || l.err != nil || len(s.notes) != 0 {
		t.Errorf("after a quiet ask failed: %q, %v, notes %v; want yesterday's, without a word", l.value, l.err, s.notes)
	}
	asks := l.asks
	s.keepFresh("daily", time.Hour)
	fetch(s, "daily", get)
	if l.asks != asks {
		t.Errorf("asked again at once after a failure")
	}

	l.askedAt = l.askedAt.Add(-freshRetry)
	fail, answer = false, "today"
	s.keepFresh("daily", time.Hour)
	fetch(s, "daily", get)
	answered(l, 2)
	if l.value != "today" {
		t.Errorf("after the retry: %q; want today's", l.value)
	}
}
