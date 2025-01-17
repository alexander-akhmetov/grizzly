package grizzly

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

type watch struct {
	path   string
	parent string
	isDir  bool
}

type Watcher struct {
	watcher     *fsnotify.Watcher
	watcherFunc func(string) error
	watches     []watch
}

func NewWatcher(watcherFunc func(path string) error) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := Watcher{
		watcher:     w,
		watcherFunc: watcherFunc,
	}
	return &watcher, nil
}

func (w *Watcher) Add(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		// `vim` renames and replaces, doesn't create a WRITE event. So we need to watch the whole dir and filter for our file
		parent := filepath.Dir(path) + "/"
		log.WithField("path", parent).Debug("[watcher] Adding path to watch list")
		w.watches = append(w.watches, watch{path: path, parent: parent, isDir: false})
		err := w.watcher.Add(parent)
		if err != nil {
			return err
		}
	} else {
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if !strings.HasSuffix(path, "/") {
					path += "/"
				}
				log.WithField("path", path).Debug("[watcher] Adding path to watch list")
				w.watches = append(w.watches, watch{path: path, parent: path, isDir: true})
				return w.watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (w *Watcher) Watch() error {
	go func() {
		log.Info("[watcher] Watching for changes")
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if w.isWatched(event.Name) {
						log.Debugf("[watcher] Changes detected: %s %s ", event.Op.String(), event.Name)
						err := w.watcherFunc(event.Name)
						if err != nil {
							log.Warn("[watcher] error: ", err)
						}
					}
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Warn("[watcher] error: ", err)
			}
		}
	}()
	return nil
}

func (w *Watcher) Wait() error {
	done := make(chan bool)
	<-done
	return nil
}

func (w *Watcher) isWatched(path string) bool {
	parent := filepath.Dir(path) + "/"
	cleanPath := filepath.Clean(path)
	for _, watchTarget := range w.watches {
		if parent != watchTarget.parent {
			continue
		}

		if watchTarget.isDir || filepath.Clean(watchTarget.path) == cleanPath {
			return true
		}
	}

	return false
}
