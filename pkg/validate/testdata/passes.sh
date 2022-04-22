#!/usr/bin/env bash

cat <<EOF
{
    "Name": "passes-bundle",
    "Errors": null,
    "Warnings": [
        {
            "Type": "CSVFileNotValid",
            "Level": "Warning",
            "Field": "",
            "BadValue": "",
            "Detail": "(gatekeeper-operator.v0.2.0-rc.3) csv.Spec.minKubeVersion is not informed. It is recommended you provide this information. Otherwise, it would mean that your operator project can be distributed and installed in any cluster version available, which is not necessarily the case for all projects."
        }
    ]
}
EOF
