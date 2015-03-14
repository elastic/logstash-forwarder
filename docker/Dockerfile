FROM debian:jessie

RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y \
  ruby \
  ruby-dev \
  golang \
  build-essential

RUN gem install bundler
