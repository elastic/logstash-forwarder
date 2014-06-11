package sodium
// #include <sodium.h>
// #cgo pkg-config: sodium
import "C"
import "unsafe"

func Randombytes(bytes []byte) {
  C.randombytes_buf(unsafe.Pointer(&bytes[0]), C.size_t(cap(bytes)))
}
