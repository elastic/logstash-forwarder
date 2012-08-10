VERSION=74
URL=http://lz4.googlecode.com/svn/trunk/
WORKDIR=lz4-r$(VERSION)

default: install
include ../Makefile.ext

fetch: 
	@# nothing to do

workdir: | $(WORKDIR)
	@# nothing to do

$(WORKDIR):
	svn export $(URL)@$(VERSION) $(WORKDIR)

configure: workdir
	@# nothing to do

build: $(WORKDIR)/liblz4.$(LIBEXT).$(VERSION)
$(WORKDIR)/liblz4.$(LIBEXT).$(VERSION): configure
	cc -shared -Wl,-soname,liblz4.$(LIBEXT).$(VERSION) -fPIC -o $@ $(WORKDIR)/lz4.c $(WORKDIR)/lz4hc.c

install: $(PREFIX)/lib/liblz4.$(LIBEXT).$(VERSION) $(PREFIX)/include/lz4.h \
         $(PREFIX)/lib/liblz4.$(LIBEXT)
$(PREFIX)/lib/liblz4.$(LIBEXT).$(VERSION): | build
	install $(WORKDIR)/liblz4.$(LIBEXT).$(VERSION) $(PREFIX)/lib
$(PREFIX)/lib/liblz4.$(LIBEXT): | $(PREFIX)/lib/liblz4.$(LIBEXT).$(VERSION)
	ln -s liblz4.$(LIBEXT).$(VERSION) $@
$(PREFIX)/include/lz4.h: | build
	install -m 644 $(WORKDIR)/lz4.h $(PREFIX)/include
$(PREFIX)/include/lz4hc.h: | build
	install -m 644 $(WORKDIR)/lz4hc.h $(PREFIX)/include
