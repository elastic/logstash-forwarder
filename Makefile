VERSION=0.0.1

CFLAGS+=-Ibuild/include 
CFLAGS+=-D_POSIX_C_SOURCE=199309 -std=c99 -Wall -Wextra -Werror -pipe
CFLAGS+=-g
CFLAGS+=-Wno-unused-function
LDFLAGS+=-pthread
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'
LIBS=-lzmq -ljemalloc -lssl -lcrypto -luuid -lz
#-llz4


FETCH=sh fetch.sh
#-lmsgpack
#-ljansson

PREFIX?=/opt/lumberjack

default: build/bin/lumberjack
include Makefile.ext

ifeq ($(UNAME),Linux)
# clock_gettime is in librt on linux.
LIBS+=-lrt
endif

clean:
	-@rm -fr lumberjack unixsock *.o build
	-@make -C vendor/msgpack/ clean
	-@make -C vendor/jansson/ clean
	-@make -C vendor/jemalloc/ clean
	-@make -C vendor/libuuid/ clean
	-@make -C vendor/zeromq/ clean

rpm deb:
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) --prefix /opt/lumberjack \
		--exclude '*.a' -C build bin/lumberjack lib

#install: build/bin/lumberjack build/lib/libzmq.$(LIBEXT)
# install -d -m 755 build/bin/* $(PREFIX)/bin/lumberjack
# install -d build/lib/* $(PREFIX)/lib

#unixsock.c: build/include/insist.h
backoff.c: backoff.h
harvester.c: harvester.h proto.h str.h build/include/insist.h build/include/zmq.h
emitter.c: emitter.h ring.h build/include/zmq.h 
lumberjack.c: build/include/insist.h build/include/zmq.h 
lumberjack.c: backoff.h harvester.h emitter.h
harvester.c lumberjack.c pushpull.c ring.c str.c: build/include/jemalloc/jemalloc.h
str.c: str.h
proto.c: proto.h str.h
ring.c: ring.h

proto.c: build/include/lz4.h

.PHONY: test
test: | build/test/test_ring
	build/test/test_ring

# Tests
test_ring.c: ring.h build/include/jemalloc/jemalloc.h build/include/insist.h
build/test/test_ring: test_ring.o ring.o  | build/test
	$(CC) $(LDFLAGS) -o $@ $^ -ljemalloc

#build/bin/pushpull: | build/lib/libzmq.$(LIBEXT) build/lib/libmsgpack.$(LIBEXT) build/bin
#build/bin/pushpull: pushpull.o
#	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)

build/bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT)
build/bin/lumberjack: | build/lib/libjemalloc.$(LIBEXT)
build/bin/lumberjack: | build/lib/libz.$(LIBEXT)
build/bin/lumberjack: | build/lib/liblz4.$(LIBEXT)
build/bin/lumberjack: lumberjack.o backoff.o harvester.o emitter.o str.o proto.o ring.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)
	@echo " => Build complete: $@"
	@echo " => Run 'make rpm' to build an rpm (or deb or tarball)"

build/include/insist.h: | build/include
	PATH=$$PWD:$$PATH fetch.sh -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build/include/zmq.h build/lib/libzmq.$(LIBEXT): | build
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zeromq/ install PREFIX=$$PWD/build

build/include/msgpack.h build/lib/libmsgpack.$(LIBEXT): | build
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/msgpack/ install PREFIX=$$PWD/build

build/include/jemalloc/jemalloc.h build/lib/libjemalloc.$(LIBEXT): | build
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/jemalloc/ install PREFIX=$$PWD/build

build/include/lz4.h build/lib/liblz4.$(LIBEXT): | build
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/lz4/ install PREFIX=$$PWD/build

build/include/zlib.h build/lib/libz.$(LIBEXT): | build
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zlib/ install PREFIX=$$PWD/build

build:
	mkdir $@

build/include build/bin build/test: | build
	mkdir $@
