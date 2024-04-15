#!/usr/bin/bash
cd config/samples

sj=false

while getopts ":s" opt; do
  case ${opt} in
    s )
      sj=true
      ;;
    \? )
      echo "Invalid Option: -$OPTARG" 1>&2
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))

jobs=$(kubectl get jobs -o name)

count=0
for job in $jobs; do
    if [[ $sj == true ]]; then
        if [[ $job =~ "sleepy-scavengerjob" ]]; then
            count=$((1+count))
        fi
    else
        if [[ $job =~ "sleep-infinity" ]]; then
            count=$((1+count))
        fi
    fi
done

for i in $(seq 1 $1)
do
    i=$((i+count))
    if [[ $sj == true ]]; then
        sed 's|COUNT|'${i}'|g' sleepy_scavengerjob.yaml > job-$i.yaml
    else
        sed 's|COUNT|'${i}'|g' sleep.yaml > job-$i.yaml
    fi
    kubectl apply -f job-$i.yaml
done
rm job-*.yaml