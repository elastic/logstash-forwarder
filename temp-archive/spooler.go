package main

import (
  "time"
)

func Spool(input chan *FileEvent,
  output chan []*FileEvent,
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
    case event := <-input:
      //append(spool, event)
      spool[spool_i] = event
      spool_i++

      // Flush if full
      if spool_i == cap(spool) {
        //spoolcopy := make([]*FileEvent, max_size)
        var spoolcopy []*FileEvent
        //fmt.Println(spool[0])
        spoolcopy = append(spoolcopy, spool[:]...)
        output <- spoolcopy
        next_flush_time = time.Now().Add(idle_timeout)

        spool_i = 0
      }
    case <-ticker.C:
      //fmt.Println("tick")
      if now := time.Now(); now.After(next_flush_time) {
        // if current time is after the next_flush_time, flush!
        //fmt.Printf("timeout: %d exceeded by %d\n", idle_timeout,
        //now.Sub(next_flush_time))

        // Flush what we have, if anything
        if spool_i > 0 {
          var spoolcopy []*FileEvent
          spoolcopy = append(spoolcopy, spool[0:spool_i]...)
          output <- spoolcopy
          next_flush_time = now.Add(idle_timeout)
          spool_i = 0
        }
      } /* if 'now' is after 'next_flush_time' */
      /* case ... */
    } /* select */
  } /* for */
} /* spool */
