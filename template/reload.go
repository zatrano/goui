package template

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const reloadDebounce = 100 * time.Millisecond

// startWatch begins recursive fsnotify watching of rootDir.
// Called only when Config.WatchForChanges is true.
func (r *Registry) startWatch() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	r.watcher = w
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})

	if err := addWatchRecursive(w, r.rootDir); err != nil {
		_ = w.Close()
		return err
	}

	go r.watchLoop()
	return nil
}

// Close stops the file watcher and background reload goroutine.
// It is a no-op when WatchForChanges was false. Safe to call multiple times.
func (r *Registry) Close() error {
	var err error
	r.closeOnce.Do(func() {
		if r.stopCh != nil {
			close(r.stopCh)
		}
		if r.watcher != nil {
			err = r.watcher.Close()
		}
		if r.doneCh != nil {
			<-r.doneCh
		}
	})
	return err
}

func (r *Registry) watchLoop() {
	defer close(r.doneCh)

	var timer *time.Timer
	var timerCh <-chan time.Time

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerCh = nil
		}
	}

	for {
		select {
		case <-r.stopCh:
			stopTimer()
			return

		case ev, ok := <-r.watcher.Events:
			if !ok {
				stopTimer()
				return
			}
			if !isReloadEvent(ev) {
				continue
			}
			// New directories created at runtime: start watching them.
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = addWatchRecursive(r.watcher, ev.Name)
				}
			}
			stopTimer()
			timer = time.NewTimer(reloadDebounce)
			timerCh = timer.C

		case err, ok := <-r.watcher.Errors:
			if !ok {
				stopTimer()
				return
			}
			r.reportReloadError(err)

		case <-timerCh:
			timer = nil
			timerCh = nil
			r.recompile()
		}
	}
}

func isReloadEvent(ev fsnotify.Event) bool {
	if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) &&
		!ev.Has(fsnotify.Rename) && !ev.Has(fsnotify.Remove) {
		return false
	}
	// React to .goui.html changes; also directory create/remove (may add/remove templates).
	name := ev.Name
	if strings.HasSuffix(name, gouiExt) {
		return true
	}
	if info, err := os.Stat(name); err == nil && info.IsDir() {
		return true
	}
	// Remove/rename: path may no longer exist; still reload if it looked like a template.
	if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		return strings.HasSuffix(name, gouiExt) || filepath.Ext(name) == ""
	}
	return false
}

func (r *Registry) recompile() {
	root, names, warnings, err := r.build()
	if err != nil {
		r.reportReloadError(err)
		return
	}
	r.mu.Lock()
	r.root = root
	r.names = names
	r.warnings = warnings
	r.mu.Unlock()

	if r.cfg.OnReload != nil {
		r.cfg.OnReload()
	}
}

func (r *Registry) reportReloadError(err error) {
	if r.cfg.OnReloadError != nil {
		r.cfg.OnReloadError(err)
		return
	}
	log.Printf("goui/template: reload error: %v", err)
}

func addWatchRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}
