VERSION=0.5.5
URL=http://msgpack.org/releases/cpp/msgpack-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=msgpack-$(VERSION)

default: install
include ../Makefile.ext

default: configure

configure: configure-autoconf

ifeq ($(UNAME),Darwin)
# Use --disable-dependency-tracking otherwise llvm/clang crashes
# on OS X 10.7.x
$(WORKDIR)/Makefile: | $(WORKDIR)
	cd $(WORKDIR); ./configure --prefix=$(PREFIX) --disable-dependency-tracking
endif

build: $(WORKDIR)/src/.libs/libmsgpack.so
$(WORKDIR)/src/.libs/libmsgpack.so: configure
	$(MAKE) -C $(WORKDIR)

install: build
	$(MAKE) -C $(WORKDIR) install
