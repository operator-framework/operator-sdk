#!/bin/bash

if [ "$KUBECONFIG" = "/kubeconfig" ]; then
    if [ "$1" = "--version=2" ]; then
        cat scorecard/assets/test-5.json
        exit 0
    fi

    cat scorecard/assets/test-4.json
    exit 0
fi


if [ "$KUBECONFIG" = "~/.kube/config2" ]; then
    cat scorecard/assets/test-3.json
    exit 0
fi

if [ "$1" = "--version=2" ]; then
    cat scorecard/assets/test-2.json
    exit 0
fi

cat scorecard/assets/test-1.json
