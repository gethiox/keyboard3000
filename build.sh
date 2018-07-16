#!/bin/bash

[ -d "./build" ] || mkdir "./build"

go build -o build/keyboard3000 .
build_status=$?

[[ ${build_status} == 0 ]] || exit 1

if [[ $1 == "run" && ${build_status} == 0 ]] || [[ $1 == "--run" && ${build_status} == 0 ]]; then
    ./build/keyboard3000
else
    if [[ $1 != '' ]]; then
        echo 'Wrong parameter'
        exit 1
    fi
fi

