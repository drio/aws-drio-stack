#!/bin/bash -e

usage() {
  cat <<EOF
Usage: $(basename $0) <env> <instance_number>

Example:
  $0 staging 2
EOF
  exit 0
}

ENV=$1
NUMBER=$2
[ ".$NUMBER" == "." ] && usage
STACK_NAME=drio-$ENV-drtufts-net

[ "$NUMBER" == "1" ] && NUMBER=""

id=$(aws cloudformation list-stack-resources \
  --stack-name=$STACK_NAME | \
  jq ".StackResourceSummaries[] | \
  select (.LogicalResourceId == \"Instance$NUMBER\") | .PhysicalResourceId" | sed 's/"//g')

aws ec2 describe-instances \
  --instance-ids $id \
  --query 'Reservations[].Instances[].PublicDnsName' | jq '.[]' | sed 's/"//g'
