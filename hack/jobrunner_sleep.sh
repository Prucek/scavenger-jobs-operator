#!/usr/bin/bash
cd config/samples
for i in $(seq 1 $1)
do
    sed 's|COUNT|'${i}'|g' sleep.yaml > job-$i.yaml
    kubectl apply -f job-$i.yaml
done
rm job-*.yaml