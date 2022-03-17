#!/bin/bash -e

#instance_name=Instance2
STACK_NAME="drio-prod-drtufts-net"
INSTANCE_NUMBER="" # 1 = "" 2 = "2"

#instance_id=$(aws ec2 describe-instances \
#    --filter "Name=tag:Name,Values=${stackname}" \
#    --query "Reservations[].Instances[?State.Name == 'running'].InstanceId[]" \
#    --output text)

id=$(aws cloudformation list-stack-resources \
  --stack-name=$STACK_NAME | \
  jq ".StackResourceSummaries[] | \
  select (.LogicalResourceId == \"Instance$NUMBER\") | .PhysicalResourceId" | sed 's/"//g')


aws ssm start-session --target $id
