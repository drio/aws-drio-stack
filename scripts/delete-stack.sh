#!/bin/bash -e


SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
source $SCRIPTPATH/common.sh

confirm
aws cloudformation delete-stack \
  --region $REGION \
  --stack-name $STACK_NAME
