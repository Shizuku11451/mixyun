package main

import "time"

// homeFresh is how old NetEase's Home may grow while it shows, before it's
// asked for again: its daily recommendations change once a day, at a time
// of NetEase's, which the answers tell. freshRetry is how long an ask
// waits after the one before, as one that failed.
const (
	homeFresh  = time.Hour
	freshRetry = 5 * time.Minute
)

// freshener is a loaded of any type, as keepFresh sees it.
type freshener interface {
	freshen(age time.Duration, now time.Time) time.Duration
}

// freshen has fetch ask again, quietly, once the answer it took is older
// than age and its last ask older than freshRetry; it tells how long until
// it is due. Until the first answer, or while another ask is out, it waits;
// what fetch never asked for, as a test's, it leaves.
func (l *loaded[T]) freshen(age time.Duration, now time.Time) time.Duration {
	if !l.done || l.askedAt.IsZero() || l.again || l.refreshing || l.quiet {
		return freshRetry
	}
	due := l.at.Add(age)
	if retry := l.askedAt.Add(freshRetry); retry.After(due) {
		due = retry
	}
	if wait := due.Sub(now); wait > 0 {
		return wait
	}
	l.again, l.quiet = true, true
	return freshRetry
}

// keepFresh has what fetch got for key asked for again once older than
// age, while the page that calls it shows: coming back to it, or as it
// stays in view, as the window wakes for it. What it showed stays until
// the answer comes, and after if it fails.
func (s *shell) keepFresh(key string, age time.Duration) {
	l, ok := s.loads[key].(freshener)
	if !ok {
		return
	}
	// By the wall's clock, which a sleep moves on.
	wait := l.freshen(age, time.Now().Round(0))
	if s.freshTimer == nil {
		s.freshTimer = time.AfterFunc(wait, func() { s.update(func() { s.freshTimer = nil }) })
	}
}
