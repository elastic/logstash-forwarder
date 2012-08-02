VERSION=2.3.1
URL=http://www.digip.org/jansson/releases/jansson-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=jansson-$(VERSION)

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

build: $(WORKDIR)/src/.libs/libjansson.$(LIBEXT)
$(WORKDIR)/src/.libs/libjansson.$(LIBEXT): configure
	$(MAKE) -C $(WORKDIR)

install: build
	$(MAKE) -C $(WORKDIR) install
