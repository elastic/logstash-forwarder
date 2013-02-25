package lumberjack

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

  fmt.Printf("Starting harvester: %s\n", h.Path)

  file := h.open()
  // TODO(sissel): Ask the registrar for the start position?

  var line uint64 = 0 // Ask registrar about the line number

  // get current offset in file
  offset, _ := file.Seek(0, os.SEEK_CUR)

  // TODO(sissel): Make the buffer size tunable at start-time
  reader := bufio.NewReaderSize(file, 16<<10) // 16kb buffer by default

  for {
    text, err := h.readline(reader)

    if err != nil {
      return
    }

    line++
    event := &FileEvent{
      Source: proto.String(h.Path),
      Offset: proto.Uint64(uint64(offset)),
      Line: proto.Uint64(line),
      Text: text,
    }
    offset += int64(len(*event.Text))

    output <- event
  } /* forever */
}

func (h *Harvester) open() *os.File {
  var file *os.File
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
  file.Seek(0, os.SEEK_END)

  return file
}

func (h *Harvester) readline(reader *bufio.Reader) (*string, error) {
  var buffer bytes.Buffer
  for {
    segment, is_partial, err := reader.ReadLine()
    if err != nil {
      // TODO(sissel): Handle the error, check io.EOF?
      // TODO(sissel): if eof and line_complete is false, don't check rotation unless a very long time has passed
      if err == io.EOF {
        time.Sleep(1 * time.Second)
        continue
      } else {
        fmt.Println(err)
        return nil, err // TODO(sissel): don't do this
      }

      // TODO(sissel): At EOF, check rotation
      // TODO(sissel): if nothing to do, sleep
    }

    // TODO(sissel): if buffer exceeds a certain length, maybe report an error condition? chop it?
    buffer.Write(segment)

    if !is_partial {
      str := new(string)
      *str = buffer.String()
      return str, nil
    }
  } /* forever read chunks */

  return nil, nil
}
