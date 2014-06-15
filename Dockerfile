FROM ubuntu:14.04

# The Basics
RUN apt-get update
RUN apt-get -y install build-essential curl

# Go 1.2.2
RUN echo "deb http://mirror.anl.gov/pub/ubuntu trusty main universe" >> /etc/apt/sources.list
RUN curl https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz | tar -C /usr/local -xzf -

# Warden runtime dependencies
RUN apt-get -y install iptables quota rsync net-tools

# Redis
RUN \
  curl -L http://download.redis.io/redis-stable.tar.gz | tar xvzf - -C /tmp && \
    cd /tmp/redis-stable && \
    make && \
    make install && \
    rm -rf /tmp/redis-stable*
