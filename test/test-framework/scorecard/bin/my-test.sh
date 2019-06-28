#!/bin/bash

if [ "$KUBECONFIG" = "/kubeconfig" ]; then
    if [ "$1" = "--version=2" ]; then
        cat assets/test-5.json
        exit 0
    fi

    cat assets/test-4.json
    exit 0
fi


if [ "$KUBECONFIG" = "~/.kube/config2" ]; then
    cat assets/test-3.json
    exit 0
fi

if [ "$1" = "--version=2" ]; then
    cat assets/test-2.json
    exit 0
fi

cat assets/test-1.json
