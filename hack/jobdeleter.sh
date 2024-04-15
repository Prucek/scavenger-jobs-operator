#!/bin/bash

pattern="${1:-infinity}"

jobs=$(kubectl get jobs -o name)

for job in $jobs; do
    if [[ $job =~ $pattern ]]; then
        kubectl delete $job
    fi
done