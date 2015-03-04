.PHONY: default
default: compile

OBJECTS=logstash-forwarder

.PHONY: compile
compile: $(OBJECTS)

logstash-forwarder:
	go build -o $@

.PHONY: clean
clean: 
	-rm $(OBJECTS)
	-rm -rf build

.PHONY: generate-init-scripts
generate-init-script:
	pleaserun --install --no-install-actions --install-prefix ./build \
		--chdir /var/lib/logstash-forwarder \
		--sysv-log-path /var/log/logstash-forwarder/ \
		--overwrite -p sysv -v lsb-3.1 $(PREFIX)/bin/logstash-forwarder -config /etc/logstash-forwarder.conf
 
build/empty: | build
	mkdir $@

.PHONY: rpm deb
deb: AFTER_INSTALL=pkg/ubuntu/after-install.sh
rpm: AFTER_INSTALL=pkg/centos/after-install.sh
rpm: BEFORE_INSTALL=pkg/centos/before-install.sh
rpm: BEFORE_REMOVE=pkg/centos/before-remove.sh
deb: AFTER_INSTALL=pkg/ubuntu/after-install.sh
deb: BEFORE_INSTALL=pkg/ubuntu/before-install.sh
deb: BEFORE_REMOVE=pkg/ubuntu/before-remove.sh
rpm deb: PREFIX=/opt/logstash-forwarder
rpm deb: VERSION=$(shell ./logstash-forwarder -version)
rpm deb: compile generate-init-script build/empty
	fpm -f -s dir -t $@ -n logstash-forwarder -v $(VERSION) \
		--architecture native \
		--replaces lumberjack \
		--description "a log shipping tool" \
		--url "https://github.com/elasticsearch/logstash-forwarder" \
		--after-install $(AFTER_INSTALL) \
		--before-install $(BEFORE_INSTALL) \
		--before-remove $(BEFORE_REMOVE) \
		--config-files /etc/logstash-forwarder.conf \
		./logstash-forwarder=$(PREFIX)/bin/ \
		./logstash-forwarder.conf.example=/etc/logstash-forwarder.conf \
		./build/etc=/ \
		./build/empty/=/var/lib/logstash-forwarder/ \
		./build/empty/=/var/log/logstash-forwarder/ \
