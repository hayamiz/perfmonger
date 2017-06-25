FROM golang:1.8

WORKDIR /app

## install packages
RUN apt-get update
RUN apt-get install -y build-essential libncurses-dev libreadline-dev libssl-dev gnuplot git

## get source code
RUN git clone https://github.com/hayamiz/perfmonger .

## install RVM
RUN curl -sSL https://get.rvm.io | bash -s stable
RUN ln -sf /bin/bash /bin/sh

## install ruby
RUN bash -l -c "rvm install 1.9.3"
RUN bash -l -c "rvm use 1.9.3 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.0.0"
RUN bash -l -c "rvm use 2.0.0 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.1.10"
RUN bash -l -c "rvm use 2.1.10 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.2.7"
RUN bash -l -c "rvm use 2.2.7 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.3.4"
RUN bash -l -c "rvm use 2.3.4 && gem install bundler && bundle"

RUN bash -l -c "rvm install 2.4.1"
RUN bash -l -c "rvm use 2.4.1 && gem install bundler && bundle"

CMD true