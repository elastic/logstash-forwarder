CFLAGS+=-Ibuild/include
#LDFLAGS+=-pthread
LDFLAGS=-Lbuild/lib -lzmq -rpath $$ORIGIN/build/lib

default: lumberjack
include Makefile.ext

unixsock.c: build/include/insist.h
lumberjack.c: build/include/insist.h backoff.h build/include/zeromq.h

lumberjack: lumberjack.o backoff.o build/lib/libzmq.$(LIBEXT)

build/include/insist.h: | build/include
	curl -s -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build/include/zeromq.h build/lib/libzmq.$(LIBEXT): | build
	$(MAKE) -C vendor/zeromq/ PREFIX=$$PWD/build

build:
	mkdir $@

build/include: | build
	mkdir $@

clean:
	-rm -fr lumberjack unixsock *.o build
