VERSION=3.0.0
URL=http://www.canonware.com/download/jemalloc/jemalloc-$(VERSION).tar.bz2
TARBALL=$(shell basename $(URL))
WORKDIR=jemalloc-$(VERSION)

ifeq ($(DEBUG),1)
CONFIGURE_FLAGS=--enable-debug
endif

default: install
include ../Makefile.ext

default: configure

configure: configure-autoconf

ifeq ($(UNAME),Darwin)
# Use --disable-dependency-tracking otherwise llvm/clang crashes
# on OS X 10.7.x
$(WORKDIR)/Makefile: | $(WORKDIR)
	cd $(WORKDIR); ./configure --prefix=$(PREFIX) --disable-dependency-tracking $(CONFIGURE_FLAGS)
endif

build: $(WORKDIR)/src/.libs/libjemalloc.$(LIBEXT)
$(WORKDIR)/src/.libs/libjemalloc.$(LIBEXT): configure
	$(MAKE) -C $(WORKDIR)

install: build
	$(MAKE) -C $(WORKDIR) install
