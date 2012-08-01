VERSION=2.2.0
URL=http://download.zeromq.org/zeromq-$(VERSION).tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=zeromq-$(VERSION)

default: install
include ../Makefile.ext

CFLAGS=-I$(PREFIX)/include
LDFLAGS=-L$(PREFIX)/lib 
CXXFLAGS=-I$(PREFIX)/include

#configure: dependencies
#dependencies: $(PREFIX)/lib/libuuid.so
#dependencies: $(PREFIX)/include/uuid/uuid.h

configure: | $(WORKDIR)/Makefile
$(WORKDIR)/Makefile: | $(WORKDIR) 
	cd $(WORKDIR); CFLAGS=$(CFLAGS) CXXFLAGS=$(CXXFLAGS) LDFLAGS=$(LDFLAGS) ./configure --prefix=$(PREFIX)

compile: | $(WORKDIR)/src/.libs/libzmq.$(LIBEXT)
$(WORKDIR)/src/.libs/libzmq.$(LIBEXT): | configure
	$(MAKE) -C $(WORKDIR) CFLAGS=$(CFLAGS) CXXFLAGS=$(CXXFLAGS) LDFLAGS=$(LDFLAGS)

install: | compile $(PREFIX)/lib $(PREFIX)/include
ifeq ($(UNAME),Darwin)
	cp $(WORKDIR)/src/.libs/libzmq.1.$(LIBEXT) $(PREFIX)/lib
	ln -sf $(PREFIX)/lib/libzmq.1.$(LIBEXT) $(PREFIX)/lib/libzmq.$(LIBEXT)
else
	cp $(WORKDIR)/src/.libs/libzmq.$(LIBEXT).1 $(PREFIX)/lib
	ln -sf $(PREFIX)/lib/libzmq.$(LIBEXT).1 $(PREFIX)/lib/libzmq.$(LIBEXT)
endif
	cp $(WORKDIR)/include/zmq.h $(PREFIX)/include
	cp $(WORKDIR)/include/zmq_utils.h $(PREFIX)/include

$(PREFIX)/lib/libuuid.$(LIBEXT) $(PREFIX)/include/uuid/uuid.h:
	$(MAKE) -C ../libuuid PREFIX=$(PREFIX) $@
