#!/usr/bin/env bash

cat <<EOF
{
  "passed": false,
  "outputs": [
    {
      "type": "info",
      "message": "found bundle: ${1}"
    },
    {
      "type": "error",
      "message": "got error"
    }
  ]
}
EOF
