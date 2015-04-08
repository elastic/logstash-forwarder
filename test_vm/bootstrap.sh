#!/bin/bash


function write_to_log_files()
{
  echo "Writing test data to log files..."
  file_list="/var/log/test1.log /var/log/test2.log /var/log/test3.log /var/log/test4.log /var/log/test1.log /var/log/some-other-file1.log"
  for f in $file_list
  do
    for x in {1..10}
    do
      DT=$(date -u '+%Y-%m-%dT%k:%M:%S%z')
      if [[ "$f" == "/var/log/some-other-file1.log" ]]
      then
        echo "$DT These messages should be EXCLUDED! #$x" >> $f
      else
        echo "$DT These messages should be included! #$x" >> $f
      fi
    done
  done
}

function create_log_files()
{
  echo "Creating test log files..."
  file_list="/var/log/test1.log /var/log/test2.log /var/log/test3.log /var/log/test4.log /var/log/test1.log /var/log/some-other-file1.log"
  for f in $file_list
  do
    touch $f
  done
}

function generate_ssl_certs()
{
  echo "Generating SSL certificates..."
  mkdir -p /etc/logstash-forwarder/ssl/
  openssl genrsa -out /etc/logstash-forwarder/ssl/server.key 1024
  openssl req -new -key /etc/logstash-forwarder/ssl/server.key -batch -out /etc/logstash-forwarder/ssl/server.csr
  openssl x509 -req -days 365 -in /etc/logstash-forwarder/ssl/server.csr -signkey /etc/logstash-forwarder/ssl/server.key -out /etc/logstash-forwarder/ssl/server.crt
}

function install_go()
{
  "Installing Go..."
  bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
  source ~/.gvm/scripts/gvm
  gvm install go1.3.3
  gvm use go1.3.3
}

function clone_lsfw_pr()
{
  echo "Cloning logstash-forwarder pull request #${PULL_REQUEST_ID}..."
  cd /tmp 
  git clone https://github.com/elastic/logstash-forwarder.git
  cd logstash-forwarder
  git fetch origin pull/$PULL_REQUEST_ID/head
  git checkout -b pullrequest FETCH_HEAD
}

function build_lsfw()
{
  echo "Building logstash-forwarder..."
  cd /tmp/logstash-forwarder
  go build && mv logstash-forwarder /usr/bin
  mkdir -p /etc/logstash-forwarder/ssl
  cp /tmp/kitchen/data/logstash-forwarder.json /etc/logstash-forwarder
  cp /tmp/kitchen/data/logstash-forwarder-init /etc/init.d/logstash-forwarder 
  chmod u+x /etc/init.d/logstash-forwarder
}

PULL_REQUEST_ID=342

if [ ! -f /root/.startup_complete ]; then 

  # Update and install necessary dependancies
  apt-get update
  apt-get install -y git curl bison ruby
  gem install busser
  gem install bundler

  # Install Go
  install_go

  # Clone the logstash-forwarder repo and checkout necessary branch/pull request
  clone_lsfw_pr
  # Build logstash-forwarder and move it to /usr/bin dir
  build_lsfw

  # Create some test log files that will be used 
  create_log_files

  # Create the SSL serts for logstash-forwarder
  generate_ssl_certs

  # Start logstash forwarder
  /etc/init.d/logstash-forwarder start
  write_to_log_files

  # Set indicator file that will prevent the whole startup sequence from being executed again
  touch /root/.startup_complete

else

  echo -e "Initial provisioning was previously completed.\nAttempting to start logstash-forwarder and writting test data to logs...\n"
  LSFW_STATUS=`/etc/init.d/logstash-forwarder status | grep "is running"`
  if [ $LSFW_STATUS -ne 0 ]
  then
    /etc/init.d/logstash-forwarder start
  fi
  # Now write data to the test files 
  write_to_log_files

fi
