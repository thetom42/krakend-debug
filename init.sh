#!/bin/bash

KRAKEND_VERSION=2.0.4

# Build sample plugin
cd ./plugin && go build -gcflags='all=-N -l' -buildmode=plugin -o krakend-debugger.so

cd ..

if [ ! -d "krakend" ]; then
    # Clone krakend-ce repository
    git clone -b v${KRAKEND_VERSION} --depth 1 https://github.com/devopsfaith/krakend-ce.git krakend
else
    cd krakend
    
    # Update repo
    git pull
    
    cd ..
fi

# Make sure all go package are there for debugging
cd krakend
go mod vendor