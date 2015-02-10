.PHONY: default
default: compile

OBJECTS=logstash-forwarder

.PHONY: compile
compile: $(OBJECTS)

logstash-forwarder:
	go build

.PHONY: clean
clean: 
	-rm $(OBJECTS)
	-rm -rf build

.PHONY: generate-init-scripts
generate-init-script:
	pleaserun --install --no-install-actions --install-prefix ./build \
		--overwrite -p sysv -v lsb-3.1 $(PREFIX)/bin/logstash-forwarder 
 
.PHONY: rpm deb
rpm deb: PREFIX=/opt/logstash-forwarder
rpm deb: VERSION=$(shell ./logstash-forwarder -version)
rpm deb: compile generate-init-script
	fpm -f -s dir -t $@ -n logstash-forwarder -v $(VERSION) \
		--architecture native \
		--replaces lumberjack \
		--description "a log shipping tool" \
		--url "https://github.com/elasticsearch/logstash-forwarder" \
		./logstash-forwarder=$(PREFIX)/bin/ \
		./build/=/
