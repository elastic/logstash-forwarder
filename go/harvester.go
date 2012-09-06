package main

import (
  "os"
  "bufio"
  "fmt"
  "time"
  "backoff"
)

func fileReader(path string) (*os.File, *bufio.Reader) {
  sleeper := backoff.NewBackoff(100 * time.Millisecond, 10 * time.Second)

  /* Try forever to open this file until successful */
  file, err := os.Open(path)
  for err != nil {
    fmt.Printf("Failed to open '%s': %s\n", path, err)
    file, err = os.Open(path) /* TODO(sissel): Check for errors */
    sleeper.Wait()
  }
  reader := bufio.NewReaderSize(file, 4096)

  return file, reader
}

func Harvest(path string, output chan FileEvent) {
  file, reader := fileReader(path)
  /* Start at the end of the file when first opening. */
  file.Seek(0, os.SEEK_END)

  sleeper := backoff.NewBackoff(100 * time.Millisecond, 10 * time.Second)
  for {
    // Try to read a line from the file
    line, _, err := reader.ReadLine()

    if err != nil {
      // Errors (EOF or otherwise) indicate we should probably
      // check for file rotation.
      if shouldReopen(file, path) {
        file.Close()
        file, reader = fileReader(path)
      } else {
        /* nothing to do, wait. */
        sleeper.Wait()
      }
      continue
    }

    // Got a successful read. Emit the event.
    sleeper.Reset()
    output <- FileEvent{path, line}
  }
} /* Harvest */

func shouldReopen(file *os.File, path string) (bool) {
  // Compare the current open file against the path to see
  // if there's been any file rotation or truncation.
  filestat, err := file.Stat()
  if err != nil {
    // Error in fstat? Odd
    fmt.Printf("error fstat'ing already open file '%s': %s\n", path, err)
    return false
  }

  pathstat, err := os.Stat(path)
  if err != nil {
    // Stat on the 'path' failed, keep the current file open
    fmt.Printf("error stat on path '%s': %s\n", path, err)
    return false
  }

  /* TODO(sissel): Check if path is a symlink and if it has changed */

  if os.SameFile(filestat, pathstat) {
    /* Same file, check for truncation */
    pos, _ := file.Seek(0, os.SEEK_CUR)
    if pos > filestat.Size() {
      // Current read position exceeds file length. File was truncated.
      fmt.Printf("File '%s' truncated.\n", path);
      return true
    }
  } else {
    // Not the same file; open the path again.
    fmt.Printf("Reopening '%s' (new inode/device)\n", path)
    return true
  }
  return false
} /* checkForRotation */

func main() {
  program := os.Args[0]

  if len(os.Args) < 2 {
    fmt.Printf("Usage: %s <file> [file2] ...\n", program)
    os.Exit(1)
  }

  events := make(chan FileEvent)
  for _, path := range os.Args[1:] {
    go Harvest(path, events)
  }

  for event := range events {
    fmt.Printf("%s: %s\n", event.path, string(event.line))
  }
} /* main */
