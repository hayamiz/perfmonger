#!/bin/bash

docker build -t go-rvm .
docker tag go-rvm hayamiz/go-rvm:wercker-env
docker push hayamiz/go-rvm:wercker-env
