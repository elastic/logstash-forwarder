package main

import (
  "bufio"
  "bytes"
  "io"
  "log"
  "os" // for File and friends
  "time"
)

type Harvester struct {
  Path       string /* the file path to harvest */
  FileConfig FileConfig
  Offset     int64
  FinishChan chan int64
  Initial bool

  file *os.File /* the file being watched */
}

func (h *Harvester) Harvest(registrar_chan chan []*FileEvent, output chan *FileEvent) {
  h.open()
  info, _ := h.file.Stat() // TODO(sissel): Check error
  defer h.file.Close()
  //info, _ := file.Stat()

  // On completion, push offset so we can continue where we left off if we relaunch on the same file
  defer func() { h.FinishChan <- h.Offset }()

  // NOTE(driskell): How would we know line number if from_beginning is false and we SEEK_END? Or would we scan,count,skip?
  var line uint64 = 0 // Ask registrar about the line number

  // get current offset in file
  offset, _ := h.file.Seek(0, os.SEEK_CUR)

  if h.Initial {
    if *from_beginning {
      log.Printf("Started harvester at position %d (requested beginning): %s\n", offset, h.Path)
    } else {
      log.Printf("Started harvester at position %d (requested end): %s\n", offset, h.Path)
    }
  } else {
    log.Printf("Started harvester at position %d (requested %d): %s\n", offset, h.Offset, h.Path)
  }

  h.Offset = offset

  // Shoot a dummy FileEvent to registrar so we get this offset stored ASAP in the state file, before any events are read
  // Otherwise, if the file doesn't change, and we restart, we won't know about it when we start up again
  // And if from-beginning is false... we'll start from the end during startup even if we started from the beginning just now
  if h.Path != "-" {
    event := &FileEvent{
      Source:   &h.Path,
      Offset:   h.Offset,
      fileinfo: &info,
    }
    registrar_chan <- []*FileEvent{event}
  }

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(h.file, 16<<10) // 16kb buffer by default

  var read_timeout = 10 * time.Second
  last_read_time := time.Now()
  for {
    text, err := h.readline(reader, read_timeout)

    if err != nil {
      if err == io.EOF {
        // timed out waiting for data, got eof.
        // Check to see if the file was truncated
        info, _ := h.file.Stat()
        if info.Size() < h.Offset {
          log.Printf("File truncated, seeking to beginning: %s\n", h.Path)
          h.file.Seek(0, os.SEEK_SET)
          h.Offset = 0
          // Shoot another dummy event to registrar to ensure we update the statefile regarding the truncation
          if h.Path != "-" {
            event := &FileEvent{
              Source:   &h.Path,
              Offset:   h.Offset,
              fileinfo: &info,
            }
            registrar_chan <- []*FileEvent{event}
          }
        } else if age := time.Since(last_read_time); age > h.FileConfig.deadtime {
          // if last_read_time was more than dead time, this file is probably
          // dead. Stop watching it.
          log.Printf("Stopping harvest of %s; last change was %v ago\n", h.Path, age)
          return
        }
        continue
      } else {
        log.Printf("Unexpected state reading from %s; error: %s\n", h.Path, err)
        return
      }
    }
    last_read_time = time.Now()

    line++
    h.Offset += int64(len(*text)) + 1 // +1 because of the line terminator
    event := &FileEvent{
      Source:   &h.Path,
      Offset:   h.Offset,
      Line:     line,
      Text:     text,
      Fields:   &h.FileConfig.Fields,
      fileinfo: &info,
    }

    output <- event // ship the new event downstream
  } /* forever */
}

func (h *Harvester) open() *os.File {
  // Special handling that "-" means to read from standard input
  if h.Path == "-" {
    h.file = os.Stdin
    return h.file
  }

  for {
    var err error
    h.file, err = open_file_no_lock(h.Path)

    if err != nil {
      // retry on failure.
      log.Printf("Failed opening %s: %s\n", h.Path, err)
      time.Sleep(5 * time.Second)
    } else {
      break
    }
  }

  // TODO(sissel): Only seek if the file is a file, not a pipe or socket.
  if h.Initial {
    // This is a new file detected during startup so calculate offset based on from-beginning
    if *from_beginning {
      h.file.Seek(0, os.SEEK_SET)
    } else {
      h.file.Seek(0, os.SEEK_END)
    }
    h.Offset = 0
  } else {
    // This is a new file detected after startup, or one that was resumed from the state file; in both cases we must obey the given offset
    // For new files offset will be 0 so this ensures we always start from the beginning on files created while we're running
    // This matches the LogStash behaviour
    h.file.Seek(h.Offset, os.SEEK_SET)
  }

  return h.file
}

func (h *Harvester) readline(reader *bufio.Reader, eof_timeout time.Duration) (*string, error) {
  var buffer bytes.Buffer
  start_time := time.Now()
  for {
    segment, is_partial, err := reader.ReadLine()

    if err != nil {
      if err == io.EOF {
        time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {
          return nil, err
        }
        continue
      } else {
        log.Println(err)
        return nil, err // TODO(sissel): don't do this?
      }
    }

    // TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
    buffer.Write(segment)

    if !is_partial {
      // If we got a full line, return the whole line.
      str := new(string)
      *str = buffer.String()
      return str, nil
    }
  } /* forever read chunks */

  return nil, nil
}
