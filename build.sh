#!/bin/bash

if [ -z "$GOPATH" ]
then echo GOPATH is not defined >&2
     exit 1
fi

target_dir=$GOPATH/bin

if [ ! -d $target_dir ]
then echo GOPATH/bin is not present >&2
     exit 2
fi

if [ ! -w $target_dir ]
then echo GOPATH/bin is not writable >&2
     exit 3
fi

source_dir=$(dirname $0)

go build -o $target_dir/dialer $source_dir/dialer/main.go
chmod +x $target_dir/dialer
