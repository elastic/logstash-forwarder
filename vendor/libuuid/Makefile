VERSION=2.17.2
# Kernel.org is the original host for this project's releases, but it is 
# currently offline due to being hacked.
URL=http://ftp.us.debian.org/debian/pool/main/u/util-linux/util-linux_$(VERSION).orig.tar.gz
TARBALL=$(shell basename $(URL))
WORKDIR=util-linux-ng-$(VERSION)

default: build
include ../Makefile.ext

UUID_DIR=$(WORKDIR)/shlibs/uuid
$(UUID_DIR)/.libs/libuuid.so: configure
	$(MAKE) -C $(WORKDIR)/shlibs/uuid

.PHONY: configure
configure: configure-mine

configure-mine: $(WORKDIR)/Makefile
$(WORKDIR)/Makefile: | $(WORKDIR)
	cd $(WORKDIR); ./configure --prefix=$(PREFIX) --without-ncurses

.PHONY: install
install: $(PREFIX)/lib/libuuid.so $(PREFIX)/include/uuid/uuid.h $(PREFIX)/lib/libuuid.so.1

build: $(UUID_DIR)/.libs/libuuid.so
	$(MAKE) -C $(UUID_DIR)

$(PREFIX)/lib/libuuid.so.1: | build $(PREFIX)/lib
	@if [ -z "$(PREFIX)" ] ; then \
		echo "No PREFIX set, can't install" ; \
		exit 1; \
	fi
	cp $(UUID_DIR)/src/.libs/$(shell basename $@) $@

$(PREFIX)/lib/libuuid.so: | $(PREFIX)/lib/libuuid.so.1
	ln -sf libuuid.so.1 $@

$(PREFIX)/include/uuid/%.h: | build $(PREFIX)/include
	@if [ -z "$(PREFIX)" ] ; then \
		echo "No PREFIX set, can't install" ; \
		exit 1; \
	fi
	mkdir -p $(shell dirname $@)
	cp $(UUID_DIR)/src/$(shell basename $@) $@
