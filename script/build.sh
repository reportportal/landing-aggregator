#!/bin/bash -e


readonly prgdir=$(cd $(dirname $0); pwd)
readonly basedir=$(cd $prgdir/..; pwd)

readonly COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
readonly BUILD_DATE=$(date +%FT%T%z)

v=$1
[[ -n "$v" ]] || read -p "Enter version: " v
if [[ -z "$v" ]] ; then
	echo "Usage: $0 [<version>]"
	exit 1
fi

go build -ldflags "-X main.Branch=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE} -X main.Version=${v}" -o bin/rpLandingInfo ./landinginfo.go
