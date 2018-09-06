#!/bin/bash

if ! [[ "$0" =~ "./gencert.sh" ]]; then
  echo "must be run from 'testdata'"
  exit 255
fi

if ! which cfssl; then
  echo "cfssl is not installed"
  exit 255
fi

cfssl gencert --initca=true ./ca-csr.json | cfssljson --bare ca -
mv ca.pem ca.crt
mv ca-key.pem ca.key
if which openssl >/dev/null; then
  openssl x509 -in ca.crt -noout -text
fi
