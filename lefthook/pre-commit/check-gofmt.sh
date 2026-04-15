#!/usr/bin/env bash
output=$(gofmt -l "$@") || exit 1
if [ -n "$output" ]; then
  echo "$output"
  exit 1
fi
exit 0
