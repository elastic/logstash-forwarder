VERSION=0.3.1

# By default, all dependencies (zeromq, etc) will be downloaded and installed
# locally. You can change this if you are deploying your own.
#VENDOR?=zeromq libsodium
VENDOR=

# Where to install to.
PREFIX?=/opt/lumberjack

FETCH=sh fetch.sh
MAKE?=make
CFLAGS+=-Ibuild/include 
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'

default: build-all
build-all: build/bin/lumberjack build/bin/lumberjack.sh
#build-all: build/bin/keygen
include Makefile.ext

clean:
	-@rm -fr build bin pkg

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

rpm deb: PREFIX=/opt/lumberjack
rpm deb: | build-all
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) \
		--exclude '*.a' --exclude 'lib/pkgconfig/zlib.pc' \
		--description "a log shipping tool" \
		--url "https://github.com/jordansissel/lumberjack" \
		build/bin/lumberjack=$(PREFIX)/bin/ build/bin/lumberjack.sh=$(PREFIX)/bin/ \
		lumberjack.init=/etc/init.d/lumberjack

# Vendor'd dependencies
# If VENDOR contains 'zeromq' download and build it.
ifeq ($(filter zeromq,$(VENDOR)),zeromq)
build/bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT)
pkg/linux_amd64/github.com/alecthomas/gozmq.a: | build/lib/libzmq.$(LIBEXT)
src/github.com/alecthomas/gozmq/zmq.go: | build/lib/libzmq.$(LIBEXT)
endif # zeromq

ifeq ($(filter libsodium,$(VENDOR)),libsodium)
build/bin/lumberjack: | build/bin build/lib/libsodium.$(LIBEXT)
build/bin/lumberjack: | build/lib/pkgconfig/sodium.pc
build/bin/keygen: | build/lib/pkgconfig/sodium.pc
build/bin/keygen: | build/bin build/lib/libsodium.$(LIBEXT)
endif # libsodium

build/bin/lumberjack.sh: lumberjack.sh | build/bin
	install -m 755 $^ $@

build/bin/lumberjack: | build/bin
	PKG_CONFIG_PATH=$$PWD/build/lib/pkgconfig \
		go build -ldflags '-r $$ORIGIN/../lib' -v -o $@
build/bin/keygen:  | build/bin
	PKG_CONFIG_PATH=$$PWD/build/lib/pkgconfig \
		go install -ldflags '-r $$ORIGIN/../lib' -o $@

# Mark these phony; 'go install' takes care of knowing how and when to rebuild.
.PHONY: build/bin/keygen build/bin/lumberjack

build/lib/pkgconfig/sodium.pc: src/sodium/sodium.pc | build/lib/pkgconfig
	cp $< $@

build/lib/pkgconfig: | build/lib
	mkdir $@
build/lib: | build
	mkdir $@

# gozmq
src/github.com/alecthomas/gozmq/zmq.go:
	go get -d github.com/alecthomas/gozmq
pkg/linux_amd64/github.com/alecthomas/gozmq.a: | build/lib/libzmq.$(LIBEXT)
pkg/linux_amd64/github.com/alecthomas/gozmq.a: src/github.com/alecthomas/gozmq/zmq.go
	PKG_CONFIG_PATH=$$PWD/build/lib/pkgconfig \
	  go install -tags zmq_3_x github.com/alecthomas/gozmq

build/include/zmq.h build/lib/libzmq.$(LIBEXT): | build/include build/lib
	@echo " => Building zeromq"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/zeromq/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build/include/sodium.h build/lib/libsodium.$(LIBEXT): | build
	@echo " => Building libsodium"
	PATH=$$PWD:$$PATH $(MAKE) -C vendor/libsodium/ install PREFIX=$$PWD/build DEBUG=$(DEBUG)

build:
	mkdir $@

build/include build/bin build/test: | build
	mkdir $@

test:
	rspec
