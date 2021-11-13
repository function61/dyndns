#!/bin/bash -eu

source /build-common.sh

COMPILE_IN_DIRECTORY="cmd/dyndns"
BINARY_NAME="dyndns"

standardBuildProcess

packageLambdaFunction
