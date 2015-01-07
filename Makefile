VERSION=0.3.1

# Where to install to.
PREFIX?=/opt/logstash-forwarder

MAKE?=make
CFLAGS+=-Ibuild/include
LDFLAGS+=-Lbuild/lib -Wl,-rpath,'$$ORIGIN/../lib'

default: build-all
build-all: build/bin/logstash-forwarder build/bin/logstash-forwarder.sh

.PHONY: go-check
go-check:
	@go version > /dev/null || (echo "Go not found. You need to install go: http://golang.org/doc/install"; false)
	@go version | grep -qE 'go version go1.3|go version go1.4' || (echo "Go version 1.3 or 1.4 required, you have a version of go that is unsupported. See http://golang.org/doc/install"; false)

clean:
	-@rm -fr build bin pkg

deps-clean:
	rm -fr src/code.google.com/
	rm -fr src/github.com/ugorji/go-msgpack
	rm -fr src/github.com/alecthomas/gozmq

rpm deb: PREFIX=/opt/logstash-forwarder
rpm deb: | build-all
	fpm -s dir -t $@ -n logstash-forwarder -v $(VERSION) \
		--replaces lumberjack \
		--exclude '*.a'\
		--description "a log shipping tool" \
		--url "https://github.com/elasticsearch/logstash-forwarder" \
		build/bin/logstash-forwarder=$(PREFIX)/bin/ \
		build/bin/logstash-forwarder.sh=$(PREFIX)/bin/ \
		logstash-forwarder.init=/etc/init.d/logstash-forwarder

build/bin/logstash-forwarder.sh: logstash-forwarder.sh | build/bin
	install -m 755 $^ $@

build/bin/logstash-forwarder: | build/bin go-check
	PKG_CONFIG_PATH=$$PWD/build/lib/pkgconfig \
		go build -ldflags '-r $$ORIGIN/../lib' -v -o $@

# Mark these phony; 'go install' takes care of knowing how and when to rebuild.
.PHONY: build/bin/logstash-forwarder

build/lib/pkgconfig: | build/lib
	mkdir $@
build/lib: | build
	mkdir $@

build:
	mkdir $@

build/include build/bin build/test: | build
	mkdir $@

test:
	rspec
