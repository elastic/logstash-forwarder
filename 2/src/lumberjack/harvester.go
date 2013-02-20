package lumberjack

import (
  "os" // for File and friends
  "time"
  proto "code.google.com/p/goprotobuf/proto"
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

  for {
    dummy := &FileEvent{
      Source: proto.String("/var/log/example"),
      Offset: proto.Uint64(0),
      Line: proto.Uint64(0),
      Text: proto.String("alskdjf laskdjf laskdfj laskdjf laskdjf lasdkfj asldkfj hello world!"),
    }
    output <- dummy
    time.Sleep(10 * time.Millisecond)
  }
}
