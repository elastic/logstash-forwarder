VERSION=0.1.0

# By default, all dependencies (zeromq, etc) will be downloaded and installed
# locally. You can change this if you are deploying your own.
VENDOR?=zeromq libsodium

# Where to install to.
PREFIX?=/opt/lumberjack

FETCH=sh fetch.sh
MAKE?=make
CFLAGS+=-Ibuild/include 
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'

default: build-all
build-all: build/bin/lumberjack build/bin/lumberjack.sh
include Makefile.ext

clean:
	-@rm -fr lumberjack unixsock *.o build
deps-clean:
	rm -fr src/code.google.com/
	rm -fr src/github.com/ugorji/go-msgpack
	rm -fr src/github.com/alecthomas/gozmq

vendor-clean:
	$(MAKE) -C vendor/apr/ clean
	$(MAKE) -C vendor/jansson/ clean
	$(MAKE) -C vendor/jemalloc/ clean
	$(MAKE) -C vendor/libsodium/ clean
	$(MAKE) -C vendor/libuuid/ clean
	$(MAKE) -C vendor/lz4/ clean
	$(MAKE) -C vendor/msgpack/ clean
	$(MAKE) -C vendor/openssl/ clean
	$(MAKE) -C vendor/zeromq/ clean
	$(MAKE) -C vendor/zlib/ clean

rpm deb: | build-all
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) --prefix /opt/lumberjack \
		--exclude '*.a' --exclude 'lib/pkgconfig/zlib.pc' -C build \
		--description "a log shipping tool" \
		--url "https://github.com/jordansissel/lumberjack" \
		bin/lumberjack bin/lumberjack.sh lib

# Vendor'd dependencies
# If VENDOR contains 'zeromq' download and build it.
ifeq ($(filter zeromq,$(VENDOR)),zeromq)
bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT)
pkg/linux_amd64/github.com/alecthomas/gozmq.a: | build/lib/libzmq.$(LIBEXT)
src/github.com/alecthomas/gozmq/zmq.go: | build/lib/libzmq.$(LIBEXT)
endif # zeromq

ifeq ($(filter libsodium,$(VENDOR)),libsodium)
bin/keygen: | build/bin build/lib/libsodium.$(LIBEXT)
endif # libsodium

build/bin/lumberjack.sh: lumberjack.sh | build/bin
	install -m 755 $^ $@

build/bin/lumberjack: | build/bin bin/lumberjack
	cp bin/lumberjack build/bin/lumberjack

bin/lumberjack: pkg/linux_amd64/github.com/alecthomas/gozmq.a
	go install -ldflags '-r $$ORIGIN/../lib' lumberjack

# gozmq
src/github.com/alecthomas/gozmq/zmq.go:
	go get -d github.com/alecthomas/gozmq
	#rm src/github.com/alecthomas/gozmq/zmq_2_*.go

pkg/linux_amd64/github.com/alecthomas/gozmq.a: src/github.com/alecthomas/gozmq/zmq.go
	PKG_CONFIG_PATH=$$PWD/build/lib/pkgconfig \
	  go install -tags zmq_3_x github.com/alecthomas/gozmq

bin/keygen:
	go install -ldflags '-r $$ORIGIN/../lib' lumberjack

build/include/zmq.h build/lib/libzmq.$(LIBEXT): | build
	@echo " => Building zeromq"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zeromq/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build:
	mkdir $@

build/include build/bin build/test: | build
	mkdir $@
