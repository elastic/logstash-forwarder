package lumberjack

import (
  "time"
  "fmt"
)

func Spooler(input chan *FileEvent, 
             output chan *EventEnvelope,
             max_size uint64,
             idle_timeout time.Duration) {
  // heartbeat periodically. If the last flush was longer than
  // 'idle_timeout' time ago, then we'll force a flush to prevent us from
  // holding on to spooled events for too long.
  ticker := time.NewTicker(idle_timeout / 2)
  spool := make([]*FileEvent, max_size)
  var spool_i int = 0

  timeout := 1 * time.Second
  start := time.Now()

  for {
    select {
      case event := <- input:
        spool[spool_i] = event
        spool_i++

        // Flush if full
        if spool_i == len(spool) { 
          output <- &EventEnvelope{Events: spool[:]}
          start = time.Now() // reset 'start' time
          spool_i = 0
        }
      case <- ticker.C:
        if duration := time.Since(start); duration > timeout {
          /* Timeout occurred */
          fmt.Printf("timeout: %d > %d\n", time.Since(start), timeout)

          // Flush what we have, if anything
          if spool_i > 0 { 
            start = time.Now()
            output <- &EventEnvelope{Events: spool[0:spool_i]}
            spool_i = 0
          }
        } /* if duration > timeout */
      /* case ... */
    } /* select */
  } /* for */
} /* spooler */
