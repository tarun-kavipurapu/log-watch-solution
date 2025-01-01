package main

import (
	"fmt"
	"log"
	"os"
	"test/ws"
	"time"
)

func logTester(filePath string, duration time.Duration) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	start := time.Now()
	for time.Since(start) < duration {
		logEntry := fmt.Sprintf("Log entry at %s \n", time.Now())
		_, err := file.WriteString(logEntry)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}

		file.Sync()

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

/**
	Two Services   Watcher-----------------------WebsocketServer
	                            Watcher----------------WebsocketServer
(logfile)-->syscall.ioNotify--->publish to logChannel----->(consume the logChannel)----->client

sending last 10  lines for every new websocket connection
(seek the end) and index-1 from there

*/

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	go func() {
		logTester("test.log", time.Second*10000)
	}()

	server := ws.NewServer()
	err := server.Run("127.0.0.1:8080")
	if err != nil {
		log.Fatal("Cannot start server: ", err)
	}

}
