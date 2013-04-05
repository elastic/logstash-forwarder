package sodium 
// #include <sodium.h>
// #cgo LDFLAGS: -lsodium
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
}

func NewSession(pk [PUBLICKEYBYTES]byte, sk [SECRETKEYBYTES]byte) (s *Session){
  s = new(Session)
  s.Public = pk
  s.Secret = sk
  s.Precompute()
  return s
}

func (s *Session) Precompute() {
  C.crypto_box_curve25519xsalsa20poly1305_ref_beforenm(
    (*C.uchar)(unsafe.Pointer(&s.k[0])),
    (*C.uchar)(unsafe.Pointer(&s.Public[0])),
    (*C.uchar)(unsafe.Pointer(&s.Secret[0])))
}

func (s *Session) Box(nonce [crypto_box_NONCEBYTES]byte, plaintext []byte) (*[]byte) {
  // XXX: ciphertext needs to be zero-padded at the start for crypto_box_ZEROBYTES
  // ZEROBYTES + len(plaintext) is ciphertext length
  ciphertext := make([]byte, len(plaintext))

  C.crypto_box_curve25519xsalsa20poly1305_ref_afternm(
    (*C.uchar)(unsafe.Pointer(&ciphertext[0])),
    (*C.uchar)(unsafe.Pointer(&plaintext[0])),
    (C.ulonglong)(len(plaintext)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  return &ciphertext
}

func (s *Session) Open(nonce [crypto_box_NONCEBYTES]byte, ciphertext []byte) (*[]byte) {
  plaintext := make([]byte, len(ciphertext))

  //crypto_box_open_afternm(m,c,clen,n,k);
  C.crypto_box_curve25519xsalsa20poly1305_ref_open_afternm(
    (*C.uchar)(unsafe.Pointer(&plaintext[0])),
    (*C.uchar)(unsafe.Pointer(&ciphertext[0])),
    (C.ulonglong)(len(ciphertext)),
    (*C.uchar)(unsafe.Pointer(&nonce[0])),
    (*C.uchar)(unsafe.Pointer(&s.k[0])))

  return &plaintext
}
