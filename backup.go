package main

import (
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/fzn0x/mixmusic/internal/app"
	"github.com/fzn0x/mixmusic/locales"
)

func init() { addSettings("general", 40, (*shell).backupSettings) }

// restartApp starts Mixyun again, to put an imported backup in place;
// tests replace it.
var restartApp = func(sys *app.System) { sys.Restart() }

// backupSettings export a backup of what the user made in Mixyun, and
// import one (#7).
func (s *shell) backupSettings(c *ui.Context) {
	s.group(c, locales.BackupGroup(), func() {
		s.row(c, locales.BackupExportTitle(), locales.BackupExportAbout(), func() {
			if s.button(c, "backup-export", locales.BackupExportButton(), false).Clicked() {
				s.exportBackup()
			}
		}, "backup export move computer")
		s.row(c, locales.BackupImportTitle(), locales.BackupImportAbout(), func() {
			if s.button(c, "backup-import", locales.BackupImportButton(), false).Clicked() {
				s.chooseBackup()
			}
		}, "backup import restore move computer")
	})
	if s.confirm(c, "import-backup", &s.importing, locales.BackupImportTitle(), locales.BackupImportConfirm(), locales.BackupImportConfirmButton()) {
		s.importBackup(s.importPath)
	}
}

// backupFilters are the files a backup is.
func backupFilters() []mygo.FileFilter {
	return []mygo.FileFilter{{Name: locales.BackupFilter(), Extensions: []string{"zip"}}}
}

// exportBackup asks where to save a backup, beside the main thread, and
// saves it there.
func (s *shell) exportBackup() {
	win := s.win
	go func() {
		path, err := saveFileDialog(mygo.SaveDialogOptions{Parent: win, Title: locales.BackupExportTitle(), DefaultPath: app.BackupName(time.Now()),
			ButtonLabel: locales.BackupExportButton(), Filters: backupFilters(), CreateDirectories: true})
		if err == nil && path == "" {
			return
		}
		if err == nil {
			err = s.svc.System.ExportBackup(path)
		}
		s.update(func() {
			if err != nil {
				s.notes = append(s.notes, noticeOf(err))
				return
			}
			s.notes = append(s.notes, notice(locales.BackupSaved(filepath.Base(path))))
		})
	}()
}

// chooseBackup asks for the backup to import, beside the main thread,
// then asks before replacing anything.
func (s *shell) chooseBackup() {
	win := s.win
	go func() {
		paths, err := openFileDialog(mygo.OpenDialogOptions{Parent: win, Title: locales.BackupImportTitle(), Filters: backupFilters()})
		s.update(func() {
			switch {
			case err != nil:
				s.notes = append(s.notes, noticeOf(err))
			case len(paths) > 0:
				// The question is asked in Settings › General, also of the
				// command palette.
				s.navigate(settingsPath("general", ""))
				s.importPath, s.importing = paths[0], true
			}
		})
	}()
}

// importBackup checks the backup at path and, when it is one, starts
// Mixyun again to put it in place; otherwise it says why, and nothing
// changes.
func (s *shell) importBackup(path string) {
	go func() {
		if _, err := s.svc.System.ImportBackup(path); err != nil {
			s.update(func() { s.notes = append(s.notes, noticeOf(err)) })
			return
		}
		restartApp(s.svc.System)
	}()
}
