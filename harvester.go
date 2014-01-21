package main

import (
  "os" // for File and friends
  "log"
  "bytes"
  "io"
  "bufio"
  "time"
)

type Harvester struct {
  Path string /* the file path to harvest */
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
    text, additional_eol_bytes, err := h.readline(reader, read_timeout)

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
      Source: &h.Path,
      Offset: offset,
      Line: line,
      Text: text,
      Fields: &h.Fields,
      fileinfo: &info,
    }
    offset += int64(len(*event.Text)) + 1 + additional_eol_bytes // +1 because of the \n line terminator

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

func (h *Harvester) readline(reader *bufio.Reader, eof_timeout time.Duration) (*string, int64, error) {
  var buffer bytes.Buffer
  var additional_eol_chars int64 = 0
  start_time := time.Now()
  for {
    // try and read a byte into our buffer
    curchar, err := reader.ReadByte()
    // if there was no error
    if err == nil {
      buffer.WriteByte(curchar)
      // if the last byte read was a \n (LF)
      if curchar == '\n' {
        // remove the \n
        buffer.ReadByte()
        // check the preceding character if it was a \r (CR)
        curchar, buffer_err := buffer.ReadByte()
        // Will get buffer_error if we have an empty line
        if buffer_err == nil {
          if curchar =='\r' {
            additional_eol_chars++
          }else{
            // put it back it's not a \r
            buffer.UnreadByte()
          }
        }
        // return our line
        str := new(string)
        *str = buffer.String()
        return str, additional_eol_chars, err
      }
    } else {
      time.Sleep(1 * time.Second)
    }
    // give up waiting for data after a certain amount of time
    if time.Since(start_time) > eof_timeout {
      return nil, 0, err
    }
  }
}
