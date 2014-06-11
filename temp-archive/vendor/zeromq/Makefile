VERSION=3.2.2
ABI=3.0.0
URL=http://download.zeromq.org/zeromq-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=zeromq-$(VERSION)

ifeq ($(DEBUG),1)
CONFIGURE_FLAGS=--enable-debug
endif

default: install
include ../Makefile.ext

CFLAGS=-I$(PREFIX)/include
LDFLAGS=-L$(PREFIX)/lib 
CXXFLAGS=-I$(PREFIX)/include

#configure: dependencies
#dependencies: $(PREFIX)/lib/libuuid.so
#dependencies: $(PREFIX)/include/uuid/uuid.h

configure: | $(WORKDIR)/Makefile
$(WORKDIR)/Makefile: | $(WORKDIR) $(PREFIX)/lib/libuuid.$(LIBEXT) $(PREFIX)/include/uuid/uuid.h
	cd $(WORKDIR); CFLAGS=$(CFLAGS) CXXFLAGS=$(CXXFLAGS) LDFLAGS=$(LDFLAGS) ./configure --prefix=$(PREFIX) $(CONFIGURE_FLAGS)

compile: | $(WORKDIR)/src/.libs/libzmq.$(LIBEXT)
$(WORKDIR)/src/.libs/libzmq.$(LIBEXT): | configure
	$(MAKE) -C $(WORKDIR) CFLAGS=$(CFLAGS) CXXFLAGS=$(CXXFLAGS) LDFLAGS=$(LDFLAGS)

install: | compile $(PREFIX)/lib $(PREFIX)/include
	$(MAKE) -C $(WORKDIR) CFLAGS=$(CFLAGS) CXXFLAGS=$(CXXFLAGS) LDFLAGS=$(LDFLAGS) install
#install: $(PREFIX)/lib/libzmq.$(LIBEXT).$(ABI) $(PREFIX)/lib/libzmq.$(LIBEXT)
#install: $(PREFIX)/include/zmq.h $(PREFIX)/include/zmq_utils.h

$(PREFIX)/lib/libzmq.$(LIBEXT).$(ABI): $(WORKDIR)/src/.libs/libzmq.$(LIBEXT)
	cp $(WORKDIR)/src/.libs/libzmq.$(LIBEXT).$(ABI) $@

$(PREFIX)/lib/libzmq.$(LIBEXT): $(PREFIX)/lib/libzmq.$(LIBEXT).$(ABI)
	ln -sf libzmq.$(LIBEXT).$(ABI) $@

$(PREFIX)/include/zmq.h:
	cp $(WORKDIR)/include/zmq.h $(PREFIX)/include

$(PREFIX)/include/zmq_utils.h:
	cp $(WORKDIR)/include/zmq_utils.h $(PREFIX)/include

$(PREFIX)/lib/libuuid.$(LIBEXT) $(PREFIX)/lib/libuuid.$(LIBEXT).1 $(PREFIX)/include/uuid/uuid.h:
	$(MAKE) -C ../libuuid PREFIX=$(PREFIX) $@
