package liblumberjack

import (
  "os" // for File and friends
  "fmt"
  "bytes"
  "io"
  "bufio"
  proto "code.google.com/p/goprotobuf/proto"
  "time"
)

type Harvester struct {
  Path string /* the file path to harvest */

  file os.File /* the file being watched */
}

func (h *Harvester) Harvest(output chan *FileEvent) {
  // TODO(sissel): Read the file
  // TODO(sissel): Emit FileEvent for each line to 'output'
  // TODO(sissel): Handle rotation
  // TODO(sissel): Sleep when there's nothing to do
  // TODO(sissel): Quit if we think the file is dead (file dev/inode changed, no data in X seconds)

  fmt.Printf("Starting harvester: %s\n", h.Path)

  file := h.open()
  defer file.Close()
  //info, _ := file.Stat()

  // TODO(sissel): Ask the registrar for the start position?
  // TODO(sissel): record the current file inode/device/etc

  var line uint64 = 0 // Ask registrar about the line number

  // get current offset in file
  offset, _ := file.Seek(0, os.SEEK_CUR)

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(file, 16<<10) // 16kb buffer by default

  var read_timeout = 10 * time.Second
  last_read_time := time.Now()
  for {
    text, err := h.readline(reader, read_timeout)

    if err != nil {
      if err == io.EOF {
        // timed out waiting for data, got eof.
        // TODO(sissel): Check to see if the file was truncated
        // TODO(sissel): if last_read_time was more than 24 hours ago
        if age := time.Since(last_read_time); age > (24 * time.Hour) {
          // This file is idle for more than 24 hours. Give up and stop harvesting.
          fmt.Printf("Stopping harvest of %s; last change was %d seconds ago\n", h.Path, age.Seconds())
          return
        }
        continue
      } else {
        fmt.Printf("Unexpected state reading from %s; error: %s\n", h.Path, err)
        return
      }
    }
    last_read_time = time.Now()

    line++
    event := &FileEvent{
      Source: proto.String(h.Path),
      Offset: proto.Uint64(uint64(offset)),
      Line: proto.Uint64(line),
      Text: text,
    }
    offset += int64(len(*event.Text)) + 1  // +1 because of the line terminator

    output <- event // ship the new event downstream
  } /* forever */
}

func (h *Harvester) open() *os.File {
  var file *os.File

  // Special handling that "-" means to read from standard input
  if h.Path == "-" {
    return os.Stdin
  } 

  for {
    var err error
    file, err = os.Open(h.Path)

    if err != nil {
      // retry on failure.
      fmt.Printf("Failed opening %s: %s\n", h.Path, err)
      time.Sleep(5 * time.Second)
    } else {
      break
    }
  }

  // TODO(sissel): In the future, use the registrary to determine where to seek.
  // TODO(sissel): Only seek if the file is a file, not a pipe or socket.
  file.Seek(0, os.SEEK_END)

  return file
}

func (h *Harvester) readline(reader *bufio.Reader, eof_timeout time.Duration) (*string, error) {
  var buffer bytes.Buffer
  start_time := time.Now()
  for {
    segment, is_partial, err := reader.ReadLine()

    if err != nil {
      // TODO(sissel): if eof and line_complete is false, don't check rotation unless a very long time has passed
      if err == io.EOF {
        time.Sleep(1 * time.Second) // TODO(sissel): Implement backoff

        // Give up waiting for data after a certain amount of time.
        // If we time out, return the error (eof)
        if time.Since(start_time) > eof_timeout {
          return nil, err
        }
        continue
      } else {
        fmt.Println(err)
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
