#!/bin/bash


go build -o build/keyboard3000 .

if [[ $1 == "run" && $? == 0 ]]; then
    ./build/keyboard3000
fi

