#!/bin/bash

set -e

docker build -t go-rvm .
docker tag go-rvm hayamiz/go-rvm:wercker-env-0.11.2
docker push hayamiz/go-rvm:wercker-env-0.11.2
