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
  Path   string /* the file path to harvest */
  Fields map[string]string
  Offset int64

  file *os.File /* the file being watched */
}

func (h *Harvester) Harvest(output chan *FileEvent) {
  if h.Offset > 0 {
    log.Printf("Starting harvester at position %d: %s\n", h.Offset, h.Path)
  } else {
    log.Printf("Starting harvester: %s\n", h.Path)
  }

  h.open()
  info, _ := h.file.Stat() // TODO(sissel): Check error
  defer h.file.Close()
  //info, _ := file.Stat()

  var line uint64 = 0 // Ask registrar about the line number

  // get current offset in file
  offset, _ := h.file.Seek(0, os.SEEK_CUR)

  log.Printf("Current file offset: %d\n", offset)

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(h.file, 16<<10) // 16kb buffer by default

  var read_timeout = 10 * time.Second
  last_read_time := time.Now()
  for {
    text, bytesread, err := h.readline(reader, read_timeout)

    if err != nil {
      if err == io.EOF {
        // timed out waiting for data, got eof.
        // Check to see if the file was truncated
        info, _ := h.file.Stat()
        if info.Size() < offset {
          log.Printf("File truncated, seeking to beginning: %s\n", h.Path)
          h.file.Seek(0, os.SEEK_SET)
          offset = 0
        } else if age := time.Since(last_read_time); age > (24 * time.Hour) {
          // if last_read_time was more than 24 hours ago, this file is probably
          // dead. Stop watching it.
          // TODO(sissel): Make this time configurable
          // This file is idle for more than 24 hours. Give up and stop harvesting.
          log.Printf("Stopping harvest of %s; last change was %d seconds ago\n", h.Path, age.Seconds())
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
    event := &FileEvent{
      Source:   &h.Path,
      Offset:   offset,
      Line:     line,
      Text:     text,
      Fields:   &h.Fields,
      fileinfo: &info,
    }
    offset += int64(bytesread)

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
    h.file, err = os.Open(h.Path)

    if err != nil {
      // retry on failure.
      log.Printf("Failed opening %s: %s\n", h.Path, err)
      time.Sleep(5 * time.Second)
    } else {
      break
    }
  }

  // TODO(sissel): Only seek if the file is a file, not a pipe or socket.
  if h.Offset > 0 {
    h.file.Seek(h.Offset, os.SEEK_SET)
  } else if *from_beginning {
    h.file.Seek(0, os.SEEK_SET)
  } else {
    h.file.Seek(0, os.SEEK_END)
  }

  return h.file
}

func (h *Harvester) readline(reader *bufio.Reader, eof_timeout time.Duration) (*string, int, error) {
  var buffer bytes.Buffer
  var is_partial bool = true
  var is_cr_present bool = false
  var bufferSize int = 0;
  start_time := time.Now()

  // Store current offset for seeking back on timeout if the line is not complete
  offset, _ := h.file.Seek(0, os.SEEK_CUR)

  for {
    segment, err := reader.ReadBytes('\n')

    if segment != nil && len(segment) > 0 {
      if segment[len(segment)-1] == '\n' {
        // Found a complete line
        is_partial = false

        // Check if also a CR present
        if len(segment) > 1 && segment[len(segment)-2] == '\r' {
          is_cr_present = true;
        }
      }
    }

    if segment != nil && len(segment) > 0 {

      // TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
      writelen,_ := buffer.Write(segment)
      bufferSize += writelen;

    }

    if err != nil {
      if err == io.EOF && is_partial {
        time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {

          // If we read a partial line then we seek back otherwise we miss this part
          if len(segment) > 0 || bufferSize > 0 {
            h.file.Seek(offset, os.SEEK_SET)
          }
          return nil, 0, err
        }
        continue
      } else {
        log.Println(err)
        return nil, 0, err // TODO(sissel): don't do this?
      }
    }

    if !is_partial {
      // If we got a full line, return the whole line without the EOL chars (CRLF or LF)
      str := new(string)
      if !is_cr_present {
        *str = buffer.String()[:bufferSize-1]
      } else {
        *str = buffer.String()[:bufferSize-2]
      }
      // bufferSize returns the str length with the EOL chars (LF or CRLF)
      return str, bufferSize, nil
    }
  } /* forever read chunks */

  return nil, 0, nil
}
