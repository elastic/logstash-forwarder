package lumberjack

import (
  "time"
  //"fmt"
)

func Spooler(input chan *FileEvent, 
             output chan *EventEnvelope,
             max_size uint64,
             idle_timeout time.Duration) {
  // heartbeat periodically. If the last flush was longer than
  // 'idle_timeout' time ago, then we'll force a flush to prevent us from
  // holding on to spooled events for too long.

  ticker := time.NewTicker(idle_timeout / 2)

  // slice for spooling into
  // TODO(sissel): use container.Ring?
  spool := make([]*FileEvent, max_size)

  // Current write position in the spool
  var spool_i int = 0

  next_flush_time := time.Now().Add(idle_timeout)
  for {
    select {
      case event := <- input:
        spool[spool_i] = event
        spool_i++

        // Flush if full
        if spool_i == len(spool) { 
          output <- &EventEnvelope{Events: spool[:]}
          next_flush_time = time.Now().Add(idle_timeout)
          spool_i = 0
        }
      case <- ticker.C:
        //fmt.Println("tick")
        if now := time.Now(); now.After(next_flush_time) {
          // if current time is after the next_flush_time, flush! 
          //fmt.Printf("timeout: %d exceeded by %d\n", idle_timeout,
                     //now.Sub(next_flush_time))

          // Flush what we have, if anything
          if spool_i > 0 { 
            next_flush_time = now.Add(idle_timeout)
            output <- &EventEnvelope{Events: spool[0:spool_i]}
            spool_i = 0
          }
        } /* if 'now' is after 'next_flush_time' */
      /* case ... */
    } /* select */
  } /* for */
} /* spooler */
