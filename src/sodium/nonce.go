package sodium
import (
  "unsafe"
)

func RandomNonceStrategy() (func() [crypto_box_NONCEBYTES]byte) {
  return func () (nonce [crypto_box_NONCEBYTES]byte) {
    Randombytes(nonce[:])
    return
  }
}

func IncrementalNonceStrategy() (func() [crypto_box_NONCEBYTES]byte) {
  var nonce [crypto_box_NONCEBYTES]byte
  Randombytes(nonce[:])

  // TODO(sissel): Make the high-8 bytes of the nonce based on current time to
  // help avoid collisions?

  return func() ([crypto_box_NONCEBYTES]byte) {
    increment(nonce[:], 1)
    return nonce
  }
}

func increment(bytes []byte, value uint64) {
  for offset, carry := 0, false; carry == true || offset == 0; offset += 8 { 
    ptr := (*uint64)(unsafe.Pointer((&bytes[offset])))
    old := *ptr
    *ptr += value
    if old > *ptr {
      // overflow, carry and continue
      value = 1
      carry = true
    } else {
      carry = false
    }
  }
}
