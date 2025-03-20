#!/usr/bin/env bash

set -e

echo "Removing unused system files to gain more disk space"
rm -fr /opt/hostedtoolcache
cd /opt
find . -maxdepth 1 -mindepth 1 '!' -path ./containerd '!' -path ./actionarchivecache '!' -path ./runner '!' -path ./runner-cache -exec rm -rf '{}' ';'