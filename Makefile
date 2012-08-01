VERSION=0.0.1

CFLAGS+=-Ibuild/include
#LDFLAGS+=-pthread
LDFLAGS=-Lbuild/lib -lzmq -rpath $$ORIGIN/build/lib

default: build/bin/lumberjack
include Makefile.ext

clean:
	-rm -fr lumberjack unixsock *.o build

rpm deb:
	fpm -s dir -t $@ -n lumberjack -v $(VERSION) --prefix /opt/lumberjack \
		bin/lumberjack build/lib

#unixsock.c: build/include/insist.h
backoff.c: backoff.h
harvester.c: harvester.h
lumberjack.c: build/include/insist.h build/include/zeromq.h
lumberjack.c: backoff.h harvester.h

build/bin/lumberjack: | build/bin build/lib/libzmq.$(LIBEXT)
build/bin/lumberjack: lumberjack.o backoff.o harvester.o
	$(CC) -o $@ $^
	echo ====
	echo "Build complete: $@"
	echo "Run 'make rpm' to build an rpm (or deb or tarball)"


build/include/insist.h: | build/include
	curl -s -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build/include/zeromq.h build/lib/libzmq.$(LIBEXT): | build
	$(MAKE) -C vendor/zeromq/ PREFIX=$$PWD/build

build:
	mkdir $@

build/include: | build
	mkdir $@

build/bin: | build
	mkdir $@
