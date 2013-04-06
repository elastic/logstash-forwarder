package main

import (
  "log"
  lumberjack "liblumberjack"
  "os"
  "time"
  "flag"
  "strings"
  "runtime/pprof"
  "sodium"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var spool_size = flag.Uint64("spool-size", 1024, "Maximum number of events to spool before a flush is forced.")
var idle_timeout = flag.Duration("idle-flush-time", 5 * time.Second, "Maximum time to wait for a full spool before flushing anyway")
var server_timeout = flag.Duration("server-timeout", 30 * time.Second, "Maximum time to wait for a request to a server before giving up and trying another.")
var servers = flag.String("servers", "", "Server (or comma-separated list of servers) to send events to. Each server can be a 'host' or 'host:port'. If the port is not specified, port 5005 is assumed. One server is chosen of the list at random, and only on failure is another server used.")
var their_public_key_path = flag.String("their-public-key", "", "the file containing the NaCl public key for the server you are talking to.")
var our_secret_key_path = flag.String("my-private-key", "", "the file containing the NaCl private key for this process to encrypt with. If none is given, one is generated at runtime.")

func read_key(path string, key []byte) (err error) {
  file, err := os.Open(path)
  if err != nil {
    return
  }

  // TODO(sissel): check length of read
  _, err = file.Read(key)
  return
}

func main() {
  flag.Parse()

  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
        log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    go func() {
      time.Sleep(60 * time.Second)
      pprof.StopCPUProfile()
      panic("done")
    }()
  }

  if *public_key_path == "" {
    log.Fatalf("No -public-key flag given")
  }

  // Turn 'host' and 'host:port' into 'tcp://host:port'
  if *servers == "" {
    log.Fatalf("No servers specified, please provide the -servers setting\n")
  }

  server_list := strings.Split(*servers, ",")
  for i, server := range server_list {
    if !strings.Contains(server, ":") {
      server_list[i] = "tcp://" + server + ":5005"
    } else {
      server_list[i] = "tcp://" + server
    }
  }

  log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

  // TODO(sissel): support flags for setting... stuff
  event_chan := make(chan *lumberjack.FileEvent, 16)
  publisher_chan := make(chan []*lumberjack.FileEvent, 1)

  paths := flag.Args()

  if len(paths) == 0 {
    log.Fatalf("No paths given. What files do you want me to watch?\n")
  }

  var public_key [sodium.PUBLICKEYBYTES]byte

  err := read_key(*public_key_path, public_key[:])
  if err != nil {
    log.Fatalf("Unable to read public key path (%s): %s\n",
               *public_key_path, err)
  }

  var secret_key [sodium.SECRETKEYBYTES]byte
  if *secret_key_path  == "" {
    log.Printf("No secret key given; generating one.")
    _, secret_key = sodium.CryptoBoxKeypair()
  } else {
    err := read_key(*secret_key_path, secret_key[:])
    if err != nil {
      log.Printf("Unable to read private key (%s): %s\n",
                 *public_key_path, err)
      log.Printf("Generating a private key now.\n")
      _, sk := sodium.CryptoBoxKeypair()
      copy(secret_key[:], sk[:])
    }
  }

  // The basic model of execution:
  // - prospector: finds files in paths/globs to harvest, starts harvesters
  // - harvester: reads a file, sends events to the spooler
  // - spooler: buffers events until ready to flush to the publisher
  // - publisher: writes to the network, notifies registrar
  // - registrar: records positions of files read
  // Finally, prospector uses the registrar information, on restart, to
  // determine where in each file to resume a harvester.

  // Prospect the globs/paths given on the command line and launch harvesters
  go lumberjack.Prospect(paths, event_chan)

  // Harvesters dump events into the spooler.
  go lumberjack.Spool(event_chan, publisher_chan, *spool_size, *idle_timeout)

  lumberjack.Publish(publisher_chan, server_list, public_key, secret_key,
                     *server_timeout)

  // TODO(sissel): publisher should send state to the registrar
  // TODO(sissel): registrar records last acknowledged positions in all files.
} /* main */
