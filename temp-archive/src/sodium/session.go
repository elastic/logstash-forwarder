package sodium 
// #include <sodium.h>
// #cgo pkg-config: sodium
import "C"
import "unsafe"
import "fmt"

type Session struct {
  // the public key of the agent who is sending you encrypted messages
  Public [PUBLICKEYBYTES]byte 

  // your secret key
  Secret [SECRETKEYBYTES]byte

  // for beforenm/afternm optimization
  k [crypto_box_BEFORENMBYTES]byte

  // The nonce generator.
  Nonce func() []byte
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

func (s *Session) Box(plaintext []byte) (ciphertext []byte, nonce []byte) {
  // XXX: ciphertext needs to be zero-padded at the start for crypto_box_ZEROBYTES
  // ZEROBYTES + len(plaintext) is ciphertext length
  ciphertext = make([]byte, crypto_box_ZEROBYTES + len(plaintext))
  nonce = s.Nonce()
  
  m := make([]byte, crypto_box_ZEROBYTES + len(plaintext))
  copy(m[crypto_box_ZEROBYTES:], plaintext)

  C.crypto_box_curve25519xsalsa20poly1305_ref_afternm(
    (*C.uchar)(unsafe.Pointer(&ciphertext[0])),
    (*C.uchar)(unsafe.Pointer(&m[0])), (C.ulonglong)(len(m)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  //fmt.Printf("ciphertext: %v\n", ciphertext)
  //fmt.Printf("ciphertext2: %v\n", ciphertext[crypto_box_BOXZEROBYTES:])
  return ciphertext[crypto_box_BOXZEROBYTES:], nonce[:]
}

func (s *Session) Open(nonce []byte, ciphertext []byte) ([]byte) {
  // This function assumes the verbatim []byte given by Session.Box() is passed
  m := make([]byte, crypto_box_BOXZEROBYTES + len(ciphertext))
  copy(m[crypto_box_BOXZEROBYTES:], ciphertext)
  plaintext := make([]byte, len(m))
  if len(nonce) != crypto_box_NONCEBYTES {
    panic(fmt.Sprintf("Invalid nonce length (%d). Expected %d\n",
                      len(nonce), crypto_box_NONCEBYTES))
  }

  C.crypto_box_curve25519xsalsa20poly1305_ref_open_afternm(
    (*C.uchar)(unsafe.Pointer(&plaintext[0])),
    (*C.uchar)(unsafe.Pointer(&m[0])), (C.ulonglong)(len(m)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  return plaintext[crypto_box_ZEROBYTES:]
}
