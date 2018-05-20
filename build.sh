#!/bin/bash

[ -d "./build" ] || mkdir "./build"

go build -o build/keyboard3000 .
[[ $? == 0 ]] || exit 1

if [[ $1 == "run" && $? == 0 ]]; then
    ./build/keyboard3000
fi

