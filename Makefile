VERSION=0.0.5

# By default, all dependencies (zeromq, etc) will be downloaded and installed
# locally. You can change this if you are deploying your own.
VENDOR?=zeromq jemalloc openssl zlib

# Where to install to.
PREFIX?=/opt/lumberjack

FETCH=sh fetch.sh

CFLAGS+=-D_POSIX_C_SOURCE=199309 -std=c99 -Wall -Wextra -Werror -pipe
CFLAGS+=-g
CFLAGS+=-Wno-unused-function
LDFLAGS+=-pthread
LIBS=-lzmq -ljemalloc -lssl -lcrypto -luuid -lz

CFLAGS+=-Ibuild/include 
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'

default: all 
all: build/bin/lumberjack build/bin/lumberjack.sh

include Makefile.ext

ifeq ($(UNAME),Linux)
# clock_gettime is in librt on linux.
LIBS+=-lrt
endif

clean:
	-@rm -fr lumberjack unixsock *.o build

vendor-clean:
	-make -C vendor/msgpack/ clean
	-make -C vendor/jansson/ clean
	-make -C vendor/jemalloc/ clean
	-make -C vendor/libuuid/ clean
	-make -C vendor/zeromq/ clean
	-make -C vendor/zlib/ clean

rpm deb: | all
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) --prefix /opt/lumberjack \
		--exclude '*.a' --exclude 'lib/pkgconfig/zlib.pc' -C build \
		--description "a log shipping tool" \
		--url "https://github.com/jordansissel/lumberjack" \
		bin/lumberjack lib

#install: build/bin/lumberjack build/lib/libzmq.$(LIBEXT)
# install -d -m 755 build/bin/* $(PREFIX)/bin/lumberjack
# install -d build/lib/* $(PREFIX)/lib

backoff.c: backoff.h
harvester.c: harvester.h proto.h str.h
emitter.c: emitter.h ring.h
lumberjack.c: backoff.h harvester.h emitter.h
str.c: str.h
proto.c: proto.h str.h 
ring.c: ring.h
harvester.c: build/include/insist.h 
lumberjack.c: build/include/insist.h 

# Vendor'd dependencies
# If VENDOR contains 'zeromq' download and build it.
ifeq ($(filter zeromq,$(VENDOR)),zeromq)
emitter.c: build/include/zmq.h 
harvester.c: build/include/zmq.h
lumberjack.c:  build/include/zmq.h 
build/bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT)
endif # zeromq

ifeq ($(filter jemalloc,$(VENDOR)),jemalloc)
harvester.c lumberjack.c ring.c str.c: build/include/jemalloc/jemalloc.h
build/bin/lumberjack: | build/lib/libjemalloc.$(LIBEXT)
endif # jemalloc

ifeq ($(filter openssl,$(VENDOR)),openssl)
proto.c: build/include/openssl/ssl.h
lumberjack.c:  build/include/openssl/ssl.h
build/bin/lumberjack: | build/lib/libssl.$(LIBEXT)
build/bin/lumberjack: | build/lib/libcrypto.$(LIBEXT)
endif # openssl

ifeq ($(filter zlib,$(VENDOR)),zlib)
proto.c: build/include/zlib.h
build/bin/lumberjack: | build/lib/libz.$(LIBEXT)
endif # zlib

.PHONY: test
test: | build/test/test_ring
	build/test/test_ring

# Tests
test_ring.c: ring.h build/include/jemalloc/jemalloc.h build/include/insist.h
build/test/test_ring: test_ring.o ring.o  | build/test
	$(CC) $(LDFLAGS) -o $@ $^ -ljemalloc

proto.o: proto.c
	$(CC) $(CFLAGS) -c -o $@ $^

build/bin/lumberjack.sh: lumberjack.sh | build/bin
	install -m 755 $^ $@

build/bin/lumberjack: | build/bin
build/bin/lumberjack: lumberjack.o backoff.o harvester.o emitter.o str.o proto.o ring.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)
	@echo " => Build complete: $@"
	@echo " => Run 'make rpm' to build an rpm (or deb or tarball)"

build/include/insist.h: | build/include
	PATH=$$PWD:$$PATH fetch.sh -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build/include/zmq.h build/lib/libzmq.$(LIBEXT): | build
	@echo " => Building zeromq"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zeromq/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/msgpack.h build/lib/libmsgpack.$(LIBEXT): | build
	@echo " => Building msgpack"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/msgpack/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/jemalloc/jemalloc.h build/lib/libjemalloc.$(LIBEXT): | build
	@echo " => Building jemalloc"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/jemalloc/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/lz4.h build/lib/liblz4.$(LIBEXT): | build
	@echo " => Building lz4"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/lz4/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/zlib.h build/lib/libz.$(LIBEXT): | build
	@echo " => Building zlib"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zlib/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/openssl/ssl.h build/lib/libssl.$(LIBEXT) build/lib/libcrypto.$(LIBEXT): | build
	@echo " => Building openssl"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/openssl install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build:
	mkdir $@

build/include build/bin build/test: | build
	mkdir $@
