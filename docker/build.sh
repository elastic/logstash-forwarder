#!/bin/bash
set -eo pipefail

# Build the builder (docker image with debian, ruby, go)
docker build -t lsf-builder docker/

# Temp name for our build image
IMAGE=lsf-build-`date +'%Y-%m-%d.%H%M%S'`

# Clean up after ourselves on exit (or if an error occurs)
cleanup() {
  (docker rm $id && docker rmi $IMAGE) 2>&1 || true
}
trap cleanup EXIT

# Copy the code into the build image
id=$(tar --exclude .git --exclude *.deb -c . | docker run -i -a stdin lsf-builder /bin/bash -c "mkdir -p /logstash-forwarder && tar -xC /logstash-forwarder")
test $(docker wait $id) -eq 0
docker commit $id $IMAGE

# Build the deb
id=$(docker run -d $IMAGE /bin/bash -c "cd /logstash-forwarder; go build -o logstash-forwarder && bundle install && make deb")
docker attach $id
test $(docker wait $id) -eq 0

# Copy the deb back out
docker commit $id $IMAGE
deb_filename=$(docker run --rm $IMAGE /bin/bash -c "ls /logstash-forwarder/*.deb")
docker cp $id:$deb_filename .

# The trap above will perform cleanup :)
