package main

// This test program was used to find a difficult bug with hashing_progress.go where the number of bytes reported did not
// match the total backup size. This problem only happened under load with tens of thousands of files. This short program was
// used to break the problem down into digestable pieces. Running it in its current state is not that useful. To find the
// problem, debug output must be used at certain point. These have been left commented out.

// The log file specfied as LOG_PATH must be formatted JSON. See cmd/main.go

// All lines in the log file containing "calcFileIndexHashes: RECEIVED" are turned into Mesg structs. Each of these lines
// contain the number of bytes written in the last report and when added together should equal the total backup size. Lines
// with matching filepaths have the BytesWritnLast added together, if this value does not match the size of the file, then a
// waring is printed to stdout. Before the bug, this

// It became clear to me what the problem was after seeing multiple lines for a single file that indicated the file write was
// complete. This message should only occur once.

// The problem was fromm using pointers instead of values for HashFile. The hash goroutine passes a pointer back to the
// calcFileIndexHashes() goroutine indicating a write has occurred. A check for "hf.SizeWritn == hf.SizeTotal" is done, if
// true then "calcFileIndexHashes: RECEIVED: FILE WRITE COMPLETE" is written to the log. In the time it took to receive the
// pointer, the io.Copy goroutine has already finished writing the file, and on the completion of full write the pointer was
// sent again, producing the second "hf.SizeWritn == hf.SizeTotal" causing the second occurrence of "FILE WRITE COMPLETE" of
// output.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	LOG_PATH  = "test/log/output.log"
	totalSize uint64
	stats     = make(map[string]*stat)
	spd       = spew.ConfigState{Indent: "\t"}
)

type Mesg struct {
	BytesWritnLast float64
	FilePath       string
	Msg            string
	Time           time.Time
	Level          string
	Size           float64
}

type stat struct {
	path      string
	size      uint64
	totalSize uint64
	hits      int
}

func getTotalBytes() {
	var messages []Mesg
	file, err := os.Open("test/log/output.log")
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if !strings.Contains(text, "Inspecting file attributes") || !strings.Contains(text, "File") {
			continue
		}
		var msg Mesg
		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	for _, msg := range messages {
		totalSize += uint64(msg.Size)
	}

}

func main() {
	getTotalBytes()

	var messages []Mesg
	file, err := os.Open(LOG_PATH)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(file)
	// re := regexp.MustCompile(`\[[\d]+\][\w\ ]+RECEIVED:\ \"(.+)\"\ BytesWritnLast:\ (\d+)\ TotalBytes:\ (\d+)`)
	for scanner.Scan() {
		if !strings.Contains(scanner.Text(), "calcFileIndexHashes: RECEIVED") {
			continue
		}
		var msg Mesg
		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	var t uint64

	for _, m := range messages {
		// fmt.Println(m.Msg, m.FilePath, uint64(m.Size), uint64(m.BytesWritnLast))
		if _, ok := stats[m.FilePath]; !ok {
			stats[m.FilePath] = &stat{path: m.FilePath, size: uint64(m.BytesWritnLast), totalSize: uint64(m.Size), hits: 1}
		} else {
			stats[m.FilePath].size += uint64(m.BytesWritnLast)
			stats[m.FilePath].hits += 1
		}
		t += uint64(m.BytesWritnLast)
	}
	for _, f := range stats {
		if f.size != f.totalSize {
			fmt.Printf("Incomplete file: %q Bytes: %d TotalSize: %d hits: %d\n",
				f.path, f.size, f.totalSize, f.hits)
		}
	}
	fmt.Println(t, totalSize)
}
