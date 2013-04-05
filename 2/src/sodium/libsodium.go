package sodium
// #include <sodium.h>
// #cgo LDFLAGS: -lsodium
import "C"
import "unsafe"

const PUBLICKEYBYTES int = 32
const SECRETKEYBYTES int = 32

const crypto_box_curve25519xsalsa20poly1305_ref_BEFORENMBYTES int = 32
const crypto_box_BEFORENMBYTES int = 32
const crypto_box_NONCEBYTES int = 24

func init() {
  C.randombytes_stir();
}

func CryptoBoxKeypair() (pk [SECRETKEYBYTES]byte, sk [PUBLICKEYBYTES]byte) {
  // From golang.org/cmd/cgo
  // """ In C, a function argument written as a fixed size array actually
  //     requires a pointer to the first element of the array. C compilers are
  //     aware of this calling convention and adjust the call accordingly, but
  //     Go cannot. In Go, you must pass the pointer to the first element
  //     explicitly: C.f(&x[0]). """
  C.crypto_box_curve25519xsalsa20poly1305_ref_keypair(
    (*C.uchar)(unsafe.Pointer(&pk[0])),
    (*C.uchar)(unsafe.Pointer(&sk[0])))
  return
}

func CryptoBox(nonce []byte, plaintext string) {
  //const unsigned char pk[crypto_box_PUBLICKEYBYTES];
  //const unsigned char sk[crypto_box_SECRETKEYBYTES];
  //const unsigned char n[crypto_box_NONCEBYTES];
  //const unsigned char m[...]; unsigned long long mlen;
  //unsigned char c[...];
  //crypto_box(output (ciphertext), input (plaintext), input_len, nonce, receiver_pub, sender_secret);
  //C.crypto_box_curve25519xsalsa20poly1305_ref
}

func CryptoOpen(nonce []byte, ciphertext string) {
  //const unsigned char pk[crypto_box_PUBLICKEYBYTES];
  //const unsigned char sk[crypto_box_SECRETKEYBYTES];
  //const unsigned char n[crypto_box_NONCEBYTES];
  //const unsigned char m[...]; unsigned long long mlen;
  //unsigned char c[...];
  //crypto_box_open(output (plaintext), input (ciphertext), input_len, nonce, receiver_pub, sender_secret);
  //C.crypto_box_curve25519xsalsa20poly1305_ref_open
}
