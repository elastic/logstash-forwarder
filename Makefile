VERSION=0.0.1

CFLAGS+=-Ibuild/include 
CFLAGS+=-D_POSIX_C_SOURCE=199309 -std=c99 -Wall -Wextra -Werror -pipe -g 
# msgpack fails to compile without this.
CFLAGS+=-Wno-unused-function
LDFLAGS+=-pthread
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'
LIBS=-lzmq
#-lmsgpack
#-ljansson

PREFIX?=/opt/lumberjack

default: build/bin/lumberjack
include Makefile.ext

clean:
	-@rm -fr lumberjack unixsock *.o build
	-@make -C vendor/msgpack/ clean
	-@make -C vendor/jansson/ clean
	-@make -C vendor/zeromq/ clean

rpm deb:
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) --prefix /opt/lumberjack \
		bin/lumberjack build/lib

#install: build/bin/lumberjack build/lib/libzmq.$(LIBEXT)
# install -d -m 755 build/bin/* $(PREFIX)/bin/lumberjack
# install -d build/lib/* $(PREFIX)/lib

#unixsock.c: build/include/insist.h
backoff.c: backoff.h
harvester.c: harvester.h proto.h str.h build/include/insist.h build/include/zmq.h
emitter.c: emitter.h build/include/zmq.h
lumberjack.c: build/include/insist.h build/include/zmq.h build/include/msgpack.h
lumberjack.c: backoff.h harvester.h emitter.h
str.c: str.h
proto.c: proto.h str.h

build/bin/pushpull: | build/lib/libzmq.$(LIBEXT) build/lib/libmsgpack.$(LIBEXT) build/bin
build/bin/pushpull: pushpull.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)

build/bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT) build/lib/libmsgpack.$(LIBEXT)
build/bin/lumberjack: lumberjack.o backoff.o harvester.o emitter.o str.o proto.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)
	@echo " => Build complete: $@"
	@echo " => Run 'make rpm' to build an rpm (or deb or tarball)"


build/include/insist.h: | build/include
	curl -s -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build/include/zmq.h build/lib/libzmq.$(LIBEXT): | build
	$(MAKE) -C vendor/zeromq/ install PREFIX=$$PWD/build

#build/include/msgpack.h build/lib/libmsgpack.$(LIBEXT): | build
#	$(MAKE) -C vendor/msgpack/ install PREFIX=$$PWD/build

build/include/msgpack.h build/lib/libmsgpack.$(LIBEXT): | build
	$(MAKE) -C vendor/msgpack/ install PREFIX=$$PWD/build

build:
	mkdir $@

build/include: | build
	mkdir $@

build/bin: | build
	mkdir $@
