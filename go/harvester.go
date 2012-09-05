package main

import (
  "os"
  "bufio"
  "fmt"
  "time"
  "backoff"
  //"./event"
)

func Harvest(path string/*, output chan event.FileEvent*/) {
  file, _ := os.Open(path) /* TODO(sissel): Check for errors */
  //file.Seek(0, os.SEEK_END)

  reader := bufio.NewReaderSize(file, 4096)
  sleeper := backoff.NewBackoff(100 * time.Millisecond, 10 * time.Second)
  for {
    line, _, err := reader.ReadLine()

    if err != nil {
      checkForRotation(file, path)
      sleeper.Wait()
    } else {
      sleeper.Reset()
      fmt.Printf("%s\n", string(line))
    }
    //output <- event.FileEvent{path_bytes, "foo"[:], line}
  }
  // TODO(sissel): Read line, push to output
  // TODO(sissel): on EOF, consider rotation conditions
  // TODO(sissel): if not rotating, sleep for a time.
}

func checkForRotation(file *os.File, path string) {
  filestat, err := file.Stat() /* TODO(sissel): Check for error */
  pathstat, err := os.Stat(path)
  if err != nil {
    /* Stat on the 'path' failed, keep the current file open */
    return
  }

  /* TODO(sissel): Check if path is a symlink and if it has changed */

  if os.SameFile(filestat, pathstat) {
    /* Same file, check for truncation */
    pos, _ := file.Seek(0, os.SEEK_CUR)
    if pos > filestat.Size() {
      // Current read position exceeds file length. File was truncated.
      file.Seek(0, os.SEEK_SET)
    }
    return
  } else {
    // Not the same file, open the path again.
    //new_file, err := os.Open(path)
  }
}

func main() {
  //events := make(chan event.FileEvent)
  //Harvest("/tmp/x", events)
  Harvest("/tmp/x")
  //for event := range events {
    //fmt.Println(event)
  //}
}
