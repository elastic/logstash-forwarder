package sodium

import "fmt"
import "testing"
import "bytes"

func ExampleBox() {
  pk, sk := CryptoBoxKeypair()
  s := NewSession(pk, sk)

  original := "hello world asldkfj alsdkfj alsdkfj alwketj alwkejt lawkejt lawketjlk j"

  // Encrypt the original text
  ciphertext, nonce := s.Box([]byte(original))

  // Decrypt the ciphertext 
  plaintext := string(s.Open(nonce, ciphertext))

  fmt.Printf("%s", plaintext)
  // Output: hello world asldkfj alsdkfj alsdkfj alwketj alwkejt lawkejt lawketjlk j
}

func TestNonceGeneration(t *testing.T) {
  // This is best effort, obviously. The nonce generator is expected to never
  // produce the same nonce twice, and the most naive test we can do is to
  // require that two nonces are not the same.
  pk, sk := CryptoBoxKeypair()
  s := NewSession(pk, sk)

  original := "hello world"
  _, nonce := s.Box([]byte(original))
  _, nonce2 := s.Box([]byte(original))

  if bytes.Equal(nonce, nonce2) {
    t.Fatal("Two Box() calls generated the same nonce")
  }
}

func BenchmarkBox(b *testing.B) {
  b.StopTimer()
  pk, sk := CryptoBoxKeypair()
  s := NewSession(pk, sk)

  original := "hello world"

  b.StartTimer()
  for i := 0; i < b.N; i++ {
    ciphertext, nonce := s.Box([]byte(original))
    //plaintext := string(s.Open(nonce, ciphertext))
    s.Open(nonce, ciphertext)
  }
}
