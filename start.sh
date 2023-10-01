#!/bin/bash

# this file is for compiling the go code or running it conveniently. We have many cmd files, so we need to select one to run.

go build -o ./bin/$1 ./cmd/$1/main.go

# if $2 == 'run', then run the program
if [ $2 = "-r" ]; then
    ./bin/$1
fi