package fsnotify

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"llm-budget-tracker/internal/service"
)

type Watcher struct {
	fd     int
	file   *os.File
	events chan service.FileWatchEvent
	errors chan error

	mu        sync.Mutex
	wdToPath  map[int]string
	closed    bool
	closeOnce sync.Once
}

func NewWatcher() (*Watcher, error) {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		fd:       fd,
		file:     os.NewFile(uintptr(fd), "inotify"),
		events:   make(chan service.FileWatchEvent, 32),
		errors:   make(chan error, 8),
		wdToPath: make(map[int]string),
	}
	go watcher.readLoop()
	return watcher, nil
}

func (w *Watcher) Add(name string) error {
	mask := uint32(syscall.IN_CREATE | syscall.IN_MODIFY | syscall.IN_MOVED_TO | syscall.IN_MOVED_FROM | syscall.IN_DELETE | syscall.IN_DELETE_SELF | syscall.IN_MOVE_SELF | syscall.IN_ATTRIB)
	wd, err := syscall.InotifyAddWatch(w.fd, name, mask)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.wdToPath[wd] = name
	return nil
}

func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}

	var err error
	w.closeOnce.Do(func() {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()
		if w.file != nil {
			err = w.file.Close()
		}
	})
	return err
}

func (w *Watcher) Events() <-chan service.FileWatchEvent {
	return w.events
}

func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := w.file.Read(buf)
		if err != nil {
			w.mu.Lock()
			closed := w.closed
			w.mu.Unlock()
			if closed {
				close(w.events)
				close(w.errors)
				return
			}
			select {
			case w.errors <- err:
			default:
			}
			return
		}

		offset := 0
		for offset < n {
			raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			offset += syscall.SizeofInotifyEvent

			nameBytes := buf[offset : offset+int(raw.Len)]
			offset += int(raw.Len)
			name := string(nameBytes)
			if idx := len(name); idx > 0 {
				for idx > 0 && name[idx-1] == 0 {
					idx--
				}
				name = name[:idx]
			}

			base := w.pathForWatchDescriptor(int(raw.Wd))
			fullPath := base
			if stringsTrimmed := name; stringsTrimmed != "" {
				fullPath = filepath.Join(base, stringsTrimmed)
			}

			op := translateMask(raw.Mask)
			if op == 0 {
				continue
			}
			select {
			case w.events <- service.FileWatchEvent{Name: fullPath, Op: op}:
			default:
				select {
				case w.errors <- fmt.Errorf("dropping file watch event for %s because the event channel is full", fullPath):
				default:
				}
			}
		}
	}
}

func (w *Watcher) pathForWatchDescriptor(wd int) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.wdToPath[wd]
}

func translateMask(mask uint32) service.FileWatchOp {
	var op service.FileWatchOp
	if mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO) != 0 {
		op |= service.FileWatchCreate
	}
	if mask&syscall.IN_MODIFY != 0 {
		op |= service.FileWatchWrite
	}
	if mask&(syscall.IN_DELETE|syscall.IN_DELETE_SELF) != 0 {
		op |= service.FileWatchRemove
	}
	if mask&(syscall.IN_MOVED_FROM|syscall.IN_MOVE_SELF) != 0 {
		op |= service.FileWatchRename
	}
	if mask&syscall.IN_ATTRIB != 0 {
		op |= service.FileWatchChmod
	}
	return op
}

var _ service.FileWatcher = (*Watcher)(nil)
