
make -C ../ build/lib/libsodium.so build/lib/libzmq.so
make -C ../ build/include/sodium/sodium.h build/include/zmq.h

export CGO_CFLAGS=-I$PWD/../build/include
export CGO_LDFLAGS=-L$PWD/../build/lib

go get github.com/alecthomas/gozmq
#go install github.com/alecthomas/gozmq
#go install sodium

echo "Building lumberjack"
go install -ldflags '-r $ORIGIN/../lib' lumberjack 
echo "Building keygen"
go install -ldflags '-r $ORIGIN/../lib' keygen
