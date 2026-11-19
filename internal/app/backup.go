package app

// Backing up and restoring (#7): a zip of what the user made in Mixyun,
// read from the data folder, with a manifest.json that says what it is.
// An import is checked whole, then kept beside the data as
// backup-pending.zip, and put in place as Mixyun starts again, before
// anything opens the files: what the app holds would otherwise be saved
// over it as it quits. What it replaces is kept in before-import.
//
// shortcut: every part goes and comes back, with no check boxes, no undo
// button and no check of the other computer's paths, when users ask to
// pick parts or undo from the app.

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fzn0x/mixmusic/internal/settings"
	"github.com/fzn0x/mixmusic/internal/store"
)

// backupParts are the files and folders of the data folder a backup holds:
// the settings, the library and its play log, where episodes were left,
// the queue, the add-ons with their storage, and the covers chosen for
// playlists. Sign-ins, caches, tools, downloads and the index of local
// files stay out.
var backupParts = []string{"settings.json", "library.json", "plays.jsonl", "positions.json", "queue.json", "addons", "mix-covers"}

const (
	backupApp      = "Mixyun"
	backupManifest = "manifest.json"
	pendingBackup  = "backup-pending.zip"
	beforeImport   = "before-import"
)

// BackupManifest is what a backup says of itself.
type BackupManifest struct {
	App     string    `json:"app"`
	Version string    `json:"version"`
	System  string    `json:"system"`
	Created time.Time `json:"created"`
	Parts   []string  `json:"parts"`
}

// ErrNotBackup is what a file that isn't a backup of Mixyun returns.
var ErrNotBackup = errors.New("this file isn't a backup of Mixyun")

// BackupName is what a backup made now is called at first.
func BackupName(now time.Time) string { return "Mixyun backup " + now.Format("2006-01-02") + ".zip" }

// ExportBackup saves a backup to path, once what waits to be saved is.
func (s *System) ExportBackup(path string) error {
	a := s.a
	if p := a.goPlayer.Load(); p != nil {
		p.Flush()
	}
	if err := a.positions.Flush(); err != nil {
		return err
	}
	if err := a.settings.Flush(); err != nil {
		return err
	}
	return writeBackup(a.opts.DataDir, path, BackupManifest{App: backupApp, Version: a.version(), System: runtime.GOOS, Created: time.Now()})
}

// writeBackup writes the backup of dataDir to dst, whole or not at all.
func writeBackup(dataDir, dst string, m BackupManifest) error {
	return store.Replace(dst, 0o644, func(f *os.File) error {
		zw := zip.NewWriter(f)
		for _, part := range backupParts {
			added := false
			err := filepath.WalkDir(filepath.Join(dataDir, part), func(p string, d fs.DirEntry, err error) error {
				if errors.Is(err, fs.ErrNotExist) && p == filepath.Join(dataDir, part) {
					return nil
				}
				// Files the stores write before renaming them in place.
				if err != nil || !d.Type().IsRegular() || strings.HasSuffix(d.Name(), ".tmp") {
					return err
				}
				rel, err := filepath.Rel(dataDir, p)
				if err != nil {
					return err
				}
				added = true
				return addFile(zw, filepath.ToSlash(rel), p)
			})
			if err != nil {
				return err
			}
			if added {
				m.Parts = append(m.Parts, part)
			}
		}
		w, err := zw.Create(backupManifest)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			return err
		}
		return zw.Close()
	})
}

func addFile(zw *zip.Writer, name, p string) error {
	src, err := os.Open(p)
	if err != nil {
		return err
	}
	defer src.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

// openBackup opens the backup at p and checks it whole: its manifest, the
// names of its files, which stay in the parts, and the JSON files, which
// must read.
func openBackup(p string) (*zip.ReadCloser, BackupManifest, error) {
	var m BackupManifest
	r, err := zip.OpenReader(p)
	if err != nil {
		return nil, m, fmt.Errorf("%w: %v", ErrNotBackup, err)
	}
	fail := func(err error) (*zip.ReadCloser, BackupManifest, error) {
		r.Close()
		return nil, m, err
	}
	found := false
	for _, f := range r.File {
		if f.Name == backupManifest {
			data, err := readEntry(f)
			if err != nil || json.Unmarshal(data, &m) != nil || m.App != backupApp {
				return fail(ErrNotBackup)
			}
			found = true
			continue
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !fs.ValidPath(f.Name) || strings.Contains(f.Name, `\`) || !inParts(f.Name) {
			return fail(fmt.Errorf("%w: it holds %q", ErrNotBackup, f.Name))
		}
		if path.Ext(f.Name) != ".json" {
			continue
		}
		data, err := readEntry(f)
		if err == nil && f.Name == "settings.json" {
			_, err = settings.Decode(data)
		} else if err == nil && !json.Valid(data) {
			err = errors.New("it isn't JSON")
		}
		if err != nil {
			return fail(fmt.Errorf("the backup's %s doesn't read: %v", f.Name, err))
		}
	}
	if !found {
		return fail(ErrNotBackup)
	}
	return r, m, nil
}

// inParts tells whether the file name, in a backup, is in one of the
// parts.
func inParts(name string) bool {
	for _, part := range backupParts {
		if name == part || strings.HasPrefix(name, part+"/") {
			return true
		}
	}
	return false
}

func readEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// ImportBackup checks the backup at p and keeps it to be put in place as
// Mixyun starts again, which the caller has it do: a file that isn't a
// whole backup of Mixyun changes nothing.
func (s *System) ImportBackup(p string) (BackupManifest, error) {
	r, m, err := openBackup(p)
	if err != nil {
		return m, err
	}
	r.Close()
	src, err := os.Open(p)
	if err != nil {
		return m, err
	}
	defer src.Close()
	return m, store.Replace(filepath.Join(s.a.opts.DataDir, pendingBackup), 0o600, func(f *os.File) error {
		_, err := io.Copy(f, src)
		return err
	})
}

// applyPendingBackup puts in place the backup an import kept in dataDir,
// before anything opens the files: what it replaces goes to before-import,
// and comes back when the backup can't be put in place whole.
func applyPendingBackup(dataDir string) error {
	pending := filepath.Join(dataDir, pendingBackup)
	if _, err := os.Stat(pending); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	defer os.Remove(pending)
	r, _, err := openBackup(pending)
	if err != nil {
		return err
	}
	defer r.Close()
	aside := filepath.Join(dataDir, beforeImport)
	if err := os.RemoveAll(aside); err != nil {
		return err
	}
	if err := os.MkdirAll(aside, 0o755); err != nil {
		return err
	}
	// undo puts back the parts moved aside, over what came of the backup.
	var moved []string
	undo := func(err error) error {
		for _, part := range moved {
			// Once every part is aside, what's in the data folder came of
			// the backup.
			if len(moved) == len(backupParts) {
				os.RemoveAll(filepath.Join(dataDir, part))
			}
			os.Rename(filepath.Join(aside, part), filepath.Join(dataDir, part))
		}
		return err
	}
	for _, part := range backupParts {
		err := os.Rename(filepath.Join(dataDir, part), filepath.Join(aside, part))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return undo(err)
		}
		moved = append(moved, part)
	}
	for _, f := range r.File {
		if f.Name == backupManifest || f.FileInfo().IsDir() {
			continue
		}
		if err := extract(f, filepath.Join(dataDir, filepath.FromSlash(f.Name))); err != nil {
			return undo(err)
		}
	}
	log.Printf("mixmusic: put the imported backup in place; what it replaced is in %s", aside)
	return nil
}

func extract(f *zip.File, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := readEntry(f)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
