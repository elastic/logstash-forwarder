CFLAGS+=-Ibuild/include
#LDFLAGS+=-pthread

default: lumberjack
unixsock.c: build/include/insist.h
lumberjack.c: build/include/insist.h backoff.h

lumberjack: lumberjack.o backoff.o

build/include/insist.h: | build/include
	curl -s -o $@ https://raw.github.com/jordansissel/experiments/master/c/better-assert/insist.h

build:
	mkdir $@

build/include: | build
	mkdir $@

