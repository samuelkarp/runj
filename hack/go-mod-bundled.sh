#!/bin/sh

packages=$(go list -m all | sed -e 's/ .*//')
dump=$(strings "$1")
for package in $packages; do
  if printf '%s' "${dump}" | grep -m 1 "${package}" >/dev/null; then
    echo "${package}"
  fi
done
