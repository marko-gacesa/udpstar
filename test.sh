#!/bin/bash
go test --race -count=1 -v -timeout 30s \
  ./... \
	| if [ "$1" != 'c' ]; then cat; else sed 's#.*PASS.*#\x1b[32;1m&\x1b[0m#;s#.*FAIL.*#\x1b[31;1m&\x1b[0m#'; fi
