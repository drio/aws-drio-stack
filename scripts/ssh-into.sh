#!/bin/bash -e

instance_name=Instance2
instance_id=$(aws ec2 describe-instances \
    --filter "Name=tag:Name,Values=${instance_name}" \
    --query "Reservations[].Instances[?State.Name == 'running'].InstanceId[]" \
    --output text)

echo $instance_id
#aws ssm start-session --target i-0a76b63e8e37f3800
