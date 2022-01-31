#!/usr/bin/env bash

cat <<EOF
{
    "Name": "fails-bundle",
    "Errors": [
        {
            "Type": "CSVFileNotValid",
            "Level": "Error",
            "Field": "",
            "BadValue": "",
            "Detail": "invalid field Pesce"
        }
    ],
    "Warnings": null
}
EOF
