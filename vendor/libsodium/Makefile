VERSION=0.3
URL=http://download.dnscrypt.org/libsodium/releases/libsodium-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=libsodium-$(VERSION)

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
	cd $(WORKDIR); ./configure --prefix=$(PREFIX) $(CONFIGURE_FLAGS)
endif

build: $(WORKDIR)/src/libsodium/.libs/libsodium.$(LIBEXT)
$(WORKDIR)/src/libsodium/.libs/libsodium.$(LIBEXT): configure
	$(MAKE) -C $(WORKDIR)

install: build
	$(MAKE) -C $(WORKDIR) install
