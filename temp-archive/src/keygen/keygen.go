package main

import (
  "sodium"
  "os"
  "log"
  "flag"
)

var output = flag.String("output", "nacl", "The output prefix to use. This will generate 'nacl.secret' and 'nacl.public' for the secret and public keys, respectively, where 'nacl' is the flag value you set.")

func main() {
  flag.Parse()

  pk, sk := sodium.CryptoBoxKeypair()

  write(*output + ".public", pk[:])
  write(*output + ".secret", sk[:])
} /* main */

func write(path string, value []byte) {
  log.Printf("Writing %s", path)
  file, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE, 0600)
  if err != nil {
    log.Fatalf("Failed to open file (%s) for writing: %s\n",
               path, err)
    return
  }

  n, err := file.Write(value)
  if err != nil {
    log.Fatalf("Failed to write to file (%s): %s\n",
               path, err)
    return
  }
  if n < len(value) {
    log.Fatalf("Failed write to file (%s). Wrote %d of %d bytes.\n",
               path, n, len(value))
    return
  }

  err = file.Close()
  if err != nil {
    log.Fatalf("Failure closing file (%s): %s\n", path, err);
    return
  }
}
