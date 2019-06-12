#!/usr/bin/env bash

set -e

# Install mercurial and bazaar
yum install -y bzr mercurial

make tidy
