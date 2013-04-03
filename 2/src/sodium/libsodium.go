package sodium
// #include <sodium.h>
// #cgo LDFLAGS: -lsodium
import "C"
import "fmt"
import "unsafe"

const PUBLICKEYBYTES int = 32
const SECRETKEYBYTES int = 32

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

}

func CryptoOpen(nonce []byte, ciphertext string) {

}

func main() {
  sk, pk := CryptoBoxKeypair()
  fmt.Printf("sk: %v\n", sk)
  fmt.Printf("pk: %v\n", pk)
}
