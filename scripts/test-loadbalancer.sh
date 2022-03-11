#!/bin/bash -e

endpoint=$1
if [ ".$endpoint" == "." ];then
  echo "Need endpoint's dns"
  exit 0
fi

(
for i in {1..10}
do 
  curl -s http://$endpoint:80  
done 
) | sort | uniq -c
