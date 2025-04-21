#!/bin/bash

/home/matin/code/go-judge/runner/build-scripts/build-docker.sh
/home/matin/code/go-judge/runner/build-scripts/build-go.sh hello-world
/home/matin/code/go-judge/runner/build-scripts/build-go.sh billion-hellos
/home/matin/code/go-judge/runner/build-scripts/build-go.sh memory-hello
/home/matin/code/go-judge/runner/build-scripts/build-go.sh sum-ints