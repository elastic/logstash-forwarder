VERSION=1.2.7
URL=http://downloads.sourceforge.net/project/libpng/zlib/$(VERSION)/zlib-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=zlib-$(VERSION)

default: install
include ../Makefile.ext

configure: | $(WORKDIR)/configure.log
$(WORKDIR)/configure.log: | $(WORKDIR)
	cd $(WORKDIR); ./configure --prefix=$(PREFIX)

compile: | $(WORKDIR)/libz.$(LIBEXT).$(VERSION)
$(WORKDIR)/libz.$(LIBEXT).$(VERSION): | configure
	$(MAKE) -C $(WORKDIR)

install: | compile $(PREFIX)/lib $(PREFIX)/include
#install: | $(PREFIX)/lib/libz.$(LIBEXT).$(VERSION)
#install: | $(PREFIX)/lib/libz.$(LIBEXT)
#install: | $(PREFIX)/include/zlib.h
install:
	$(MAKE) -C $(WORKDIR) install

$(PREFIX)/lib/libz.$(LIBEXT).$(VERSION): $(WORKDIR)/libz.$(LIBEXT).$(VERSION)
	cp $^ $@

$(PREFIX)/lib/libz.$(LIBEXT): $(PREFIX)/lib/libz.$(LIBEXT).$(VERSION)
	ln -sf libz.$(LIBEXT).$(VERSION) $(PREFIX)/lib/libz.$(LIBEXT)

$(PREFIX)/include/zlib.h:
	cp $(WORKDIR)/zlib.h $(PREFIX)/include
