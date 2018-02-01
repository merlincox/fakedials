#!/bin/bash

TEMP_DIR=$(mktemp -d)
SOURCE_DIR=$(dirname $0)
TARGET_DIR=/usr/local/bin

go build -o $TEMP_DIR/dialer $SOURCE_DIR/dialer/main.go
sudo mv -f $TEMP_DIR/dialer $TARGET_DIR
sudo chmod +x $TARGET_DIR/dialer
