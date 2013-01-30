# This build actually fails under Fedora 16. I looked at redhat and debian,
# both apply like 50 patches to openssl prior to packaging.
VERSION=1.0.1c
URL=http://ftp.openssl.org/source/openssl-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=openssl-$(VERSION)

default: install
include ../Makefile.ext

default: configure

ifeq ($(DEBUG),1)
CONFIGFLAGS=-d
endif


#configure: #configure-autoconf
configure: | $(WORKDIR)
	(cd $(WORKDIR); ./config no-zlib no-krb5 enable-tlsext enable-camellia enable-mdc2 shared threads --prefix=$(PREFIX) --openssldir=$(PREFIX)/etc/ssl $(CONFIGFLAGS))

ifeq ($(UNAME),Darwin)
# Use --disable-dependency-tracking otherwise llvm/clang crashes
# on OS X 10.7.x
$(WORKDIR)/Makefile: | $(WORKDIR)
	cd $(WORKDIR); ./configure --prefix=$(PREFIX) --disable-dependency-tracking
endif

compile: $(WORKDIR)/libssl.so.1.0.0
$(WORKDIR)/libssl.so.1.0.0: | configure
	$(MAKE) -C $(WORKDIR) all

install: compile
	@# make 'all' and 'install_sw' because 'make install' installs the docs,
	@#which I don't want.
	$(MAKE) -C $(WORKDIR) install_sw

