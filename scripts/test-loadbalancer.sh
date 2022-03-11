#!/bin/bash -e

default="https://staging.drtufts.net"
endpoint=$1
if [ ".$endpoint" == "." ];then
  echo "No endpoint passed. Using $default" 
  endpoint=$default
fi

(
for i in {1..10}
do 
  curl -s $endpoint
  echo ""
done 
) | sort | uniq -c
