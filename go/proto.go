package main

import (
  "encoding/binary"
  "io"
)

func writeData(writable io.Writer, sequence uint32, m map[string]string) {
  writable.Write([]byte("1D"))
  binary.Write(writable, binary.BigEndian, uint32(sequence))
  writeMap(writable, m)
}

func writeMap(writable io.Writer, m map[string]string) {
  // How many fields in this data frame
  binary.Write(writable, binary.BigEndian, uint32(len(m)))

  for key, value := range m {
    binary.Write(writable, binary.BigEndian, uint32(len(key)))
    writable.Write([]byte(key))

    binary.Write(writable, binary.BigEndian, uint32(len(value)))
    writable.Write([]byte(value))
  }
} /* WriteMap */
