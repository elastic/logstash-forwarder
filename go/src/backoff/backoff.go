package backoff

import (
  "time"
)

type Backoff struct {
  min time.Duration
  max time.Duration
  cur time.Duration
}

func NewBackoff(min time.Duration, max time.Duration) (*Backoff) {
  return &Backoff{min, max, min}
}

func (b *Backoff) Wait() {
  time.Sleep(b.cur)

  b.cur *= 2
  if (b.cur > b.max) {
    b.cur = b.max
  }
}

func (b *Backoff) Reset() {
  b.cur = b.min
}
