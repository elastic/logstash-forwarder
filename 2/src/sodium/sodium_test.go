package sodium

import "fmt"
//import "testing"

func ExampleBoxOpen() {
  pk, sk := CryptoBoxKeypair()
  s := NewSession(pk, sk)

  var nonce [24]byte
  Randombytes(nonce[:])

  original := "hello world laksdjf laksjdf laksjdf laksjd flkasjdf laskdjf "
  ciphertext := s.Box(nonce, []byte(original))
  fmt.Println(ciphertext)
  plaintext := s.Open(nonce, *ciphertext)

  fmt.Println(plaintext)
  // Output: hello world
}
