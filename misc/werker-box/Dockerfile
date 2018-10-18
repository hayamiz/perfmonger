FROM golang:1.8

WORKDIR /app

## install packages
RUN apt-get update
RUN apt-get install -y build-essential libncurses-dev libreadline-dev libssl-dev gnuplot git gnupg2

## get source code
RUN git clone https://github.com/hayamiz/perfmonger .

## install RVM
RUN gpg --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN curl -O https://raw.githubusercontent.com/rvm/rvm/master/binscripts/rvm-installer
RUN curl -O https://raw.githubusercontent.com/rvm/rvm/master/binscripts/rvm-installer.asc
RUN gpg --verify rvm-installer.asc
RUN bash rvm-installer stable
RUN ln -sf /bin/bash /bin/sh

## install ruby
RUN bash -l -c "rvm install 2.2.10"
RUN bash -l -c "rvm use 2.2.10 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.3.8"
RUN bash -l -c "rvm use 2.3.8 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.4.5"
RUN bash -l -c "rvm use 2.4.5 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.5.3"
RUN bash -l -c "rvm use 2.5.3 && gem install bundler && bundle"

CMD true