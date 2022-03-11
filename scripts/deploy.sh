#!/bin/bash -e

AWS_ACCOUNT_ID=`aws sts get-caller-identity --query "Account" --output text`
STACK_NAME=drio-apps-$(openssl rand -hex 12)
REGION=us-east-2
EC2_INSTANCE_TYPE=t2.small
DOMAIN=drtufts.net
SUBDOMAIN=staging

#GH_ACCESS_TOKEN=$(cat .github/token)
#GH_OWNER=$(cat .github/owner)
#GH_REPO=$(cat .github/repo)
#GH_BRANCH=master

TEMPLATE=./templates/main.yml
if [ ! -f $TEMPLATE ]; then echo "Template does not exist"
  exit 1
fi

echo "stack_name: $STACK_NAME"
# Deploys static resources
# echo -e "\n\n=========== Deploying setup.yml ==========="
# aws cloudformation deploy \
#   --region $REGION \
#   --stack-name $STACK_NAME-setup \
#   --template-file setup.yml \
#   --no-fail-on-empty-changeset \
#   --capabilities CAPABILITY_NAMED_IAM

# Deploy the CloudFormation template
echo -e "\n\n=========== Deploying main.yml ==========="
aws cloudformation deploy \
  --region $REGION \
  --stack-name $STACK_NAME \
  --template-file $TEMPLATE \
  --no-fail-on-empty-changeset \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides EC2InstanceType=$EC2_INSTANCE_TYPE \
    Domain=$DOMAIN \
    SubDomain=$SUBDOMAIN

# If the deploy succeeded, show the DNS name of the created instance
if [ $? -eq 0 ]; then
  aws cloudformation list-exports \
    --query "Exports[?starts_with(Name,'InstanceEndpoint')].Value"
  aws cloudformation list-exports \
    --query "Exports[?starts_with(Name,'LBEndpoint')].Value"
fi

