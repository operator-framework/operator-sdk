#!/usr/bin/env bash

cat <<EOF
{
  "passed": true,
  "outputs": [
    {
      "type": "info",
      "message": "found bundle: ${1}"
    },
    {
      "type": "warning",
      "message": "foo"
    }
  ]
}
EOF
