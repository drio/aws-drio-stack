#!/bin/bash -e

SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
source $SCRIPTPATH/common.sh

AWS_ACCOUNT_ID=`aws sts get-caller-identity --query "Account" --output text`
CERT=`aws acm list-certificates --region $REGION --output text --query "CertificateSummaryList[?DomainName=='$DOMAIN'].CertificateArn | [0]"`

TEMPLATE=./templates/main.yml
if [ ! -f $TEMPLATE ]; then echo "Template does not exist"
  exit 1
fi

cat $TEMPLATE | sed "s/XRANDX/$RANDOM/g" > $I_TEMPLATE
trap 'rm -f -- "$I_TEMPLATE"' EXIT

# Deploy the CloudFormation template
echo -e "\n\n=========== Deploying main.yml ==========="
aws cloudformation deploy \
  --region $REGION \
  --stack-name $STACK_NAME \
  --template-file $I_TEMPLATE \
  --no-fail-on-empty-changeset \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides EC2InstanceType=$EC2_INSTANCE_TYPE \
    Domain=$DOMAIN \
    SubDomain=$SUBDOMAIN \
    Certificate=$CERT

# If the deploy succeeded, show the DNS name of the created instance
if [ $? -eq 0 ]; then
  aws cloudformation list-exports \
    --query "Exports[?starts_with(Name,'InstanceEndpoint')].Value"
  aws cloudformation list-exports \
    --query "Exports[?starts_with(Name,'LBEndpoint')].Value"
fi
