package sodium 
// #include <sodium.h>
// #cgo LDFLAGS: -lsodium
import "C"
import "unsafe"

type Session struct {
  // the public key of the agent who is sending you encrypted messages
  Public [PUBLICKEYBYTES]byte 

  // your secret key
  Secret [SECRETKEYBYTES]byte

  // for beforenm/afternm optimization
  k [crypto_box_BEFORENMBYTES]byte

  // The nonce generator.
  nonce func() [crypto_box_NONCEBYTES]byte
}

func NewSession(pk [PUBLICKEYBYTES]byte, sk [SECRETKEYBYTES]byte) (s *Session){
  s = new(Session)
  s.Public = pk
  s.Secret = sk
  s.Precompute()
  s.Nonce = RandomNonceStrategy()
  //s.Nonce = IncrementalNonceStrategy()
  return s
}

func (s *Session) Precompute() {
  C.crypto_box_curve25519xsalsa20poly1305_ref_beforenm(
    (*C.uchar)(unsafe.Pointer(&s.k[0])),
    (*C.uchar)(unsafe.Pointer(&s.Public[0])),
    (*C.uchar)(unsafe.Pointer(&s.Secret[0])))
}

func (s *Session) Box(plaintext []byte) (ciphertext []byte, nonce [crypto_box_NONCEBYTES]byte) {
  // XXX: ciphertext needs to be zero-padded at the start for crypto_box_ZEROBYTES
  // ZEROBYTES + len(plaintext) is ciphertext length
  ciphertext = make([]byte, crypto_box_ZEROBYTES + len(plaintext))
  nonce = s.nonce()

  m := make([]byte, crypto_box_ZEROBYTES + len(plaintext))
  copy(m[crypto_box_ZEROBYTES:], plaintext)

  C.crypto_box_curve25519xsalsa20poly1305_ref_afternm(
    (*C.uchar)(unsafe.Pointer(&ciphertext[0])),
    (*C.uchar)(unsafe.Pointer(&m[0])), (C.ulonglong)(len(m)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  return ciphertext, nonce
}

func (s *Session) Open(nonce [crypto_box_NONCEBYTES]byte, ciphertext []byte) ([]byte) {
  // This function assumes the verbatim []byte given by Session.Box() is passed
  plaintext := make([]byte, crypto_box_ZEROBYTES + len(ciphertext))

  C.crypto_box_curve25519xsalsa20poly1305_ref_open_afternm(
    (*C.uchar)(unsafe.Pointer(&plaintext[0])),
    (*C.uchar)(unsafe.Pointer(&ciphertext[0])), (C.ulonglong)(len(ciphertext)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  return plaintext[crypto_box_ZEROBYTES:len(ciphertext)]
}
