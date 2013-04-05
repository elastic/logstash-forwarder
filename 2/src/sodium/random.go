package sodium
// #include <sodium.h>
// #cgo LDFLAGS: -lsodium
import "C"
import "unsafe"

func Randombytes(bytes []byte) {
  C.randombytes_buf(unsafe.Pointer(&bytes[0]), C.size_t(cap(bytes)))
}
