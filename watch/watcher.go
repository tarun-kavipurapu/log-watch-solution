package watch

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	logChanInstance chan []byte
	onceLogChan     sync.Once
)

var (
	watcherInstance *Watcher
	onceWatcher     sync.Once
)

func GetLogChan() chan []byte {
	onceLogChan.Do(func() {
		logChanInstance = make(chan []byte, 256) // Buffered channel
	})
	return logChanInstance
}

func GetWatcher() (*Watcher, error) {
	onceWatcher.Do(func() {
		var err error
		watcherInstance, err = NewWatcher("test.log")
		if err != nil {
			panic(fmt.Sprintf("Failed to create Watcher: %v", err))
		}
	})
	return watcherInstance, nil
}

type Watcher struct {
	fd         int
	filePath   string
	watchDesc  int
	isWatching bool
}

func NewWatcher(filePath string) (*Watcher, error) {
	fd, err := syscall.InotifyInit()
	if err != nil {
		return nil, fmt.Errorf("inotify_init: %v", err)
	}

	watchDesc, err := syscall.InotifyAddWatch(fd, filePath, unix.IN_MODIFY)
	if err != nil {
		return nil, fmt.Errorf("inotify_add: %v", err)
	}

	return &Watcher{
		fd:         fd,
		filePath:   filePath,
		watchDesc:  watchDesc,
		isWatching: true,
	}, nil
}

func (w *Watcher) WatchFile() error {
	if !w.isWatching {
		return fmt.Errorf("no file is being watched")
	}
	file, err := os.Open(w.filePath)
	if err != nil {
		return fmt.Errorf("no file is Opened")
	}

	// Seek to the end of the file to read new lines as they are written
	file.Seek(0, io.SeekEnd)

	eventBuf := make([]byte, unix.SizeofInotifyEvent*10)
	for {
		n, err := unix.Read(w.fd, eventBuf)
		if err != nil {
			return fmt.Errorf("failed to read inotify events: %w", err)
		}

		for offset := 0; offset < n; {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&eventBuf[offset]))
			if event.Mask&unix.IN_MODIFY != 0 {
				w.printLines(file)
			}
			offset = offset + unix.SizeofInotifyEvent + int(event.Len)
		}
	}
}
func getNLine(filepath string, n int) ([]string, error) {
	fileHandle, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer fileHandle.Close()

	var (
		lines      []string
		line       string
		cursor     int64 = 0
		charBuffer       = make([]byte, 1)
	)

	stat, err := fileHandle.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}
	filesize := stat.Size()

	for {
		cursor = cursor - 1

		_, err := fileHandle.Seek(cursor, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}
		_, err = fileHandle.Read(charBuffer)
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}

		if charBuffer[0] == '\n' || charBuffer[0] == '\r' {
			if line != "" {
				lines = append([]string{line}, lines...)
				line = ""
				if len(lines) > n {
					break
				}
			}
		} else {
			line = fmt.Sprintf("%s%s", string(charBuffer), line)
		}

		if cursor == -filesize {
			lines = append([]string{line}, lines...)
			break
		}
	}

	return lines, nil
}

func (w *Watcher) SendbottomLines() error {
	lines, err := getNLine(w.filePath, 10)
	if err != nil {
		return fmt.Errorf("failed to get last 10 lines: %w", err)
	}

	logChan := GetLogChan()
	for _, line := range lines {
		logChan <- []byte(line)
	}

	return nil
}

func (w *Watcher) printLines(file *os.File) {
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil {
			return
		}
		// Insert into the channel
		logChan := GetLogChan()
		// log.Println(string(buf[:n]))
		logChan <- buf[:n]
	}
}
