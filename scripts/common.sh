#!/bin/bash -e

REGION=us-east-2
EC2_INSTANCE_TYPE=t2.small


confirm() {
    read -r -p "${1:-Are you sure? [y/N]} " response
    case "$response" in
        [yY][eE][sS]|[yY]) 
            true
            ;;
        *)
            false
            ;;
    esac
}

usage() {
  cat <<EOF
Usage: $(basename $0) <domain> <subdomain>

Example:
  $0 drtufts.net staging
EOF
  exit 0
}


#DOMAIN=drtufts.net
#SUBDOMAIN=staging
DOMAIN=$1
SUBDOMAIN=$2
[ ".$DOMAIN" == "." ] && usage
[ ".$SUBDOMAIN" == "." ] && usage

STACK_NAME=`echo "drio-$SUBDOMAIN-$DOMAIN" | gtr "." "-"`

RAND="$RANDOM-$SUBDOMAIN-$DOMAIN"
I_TEMPLATE=./template.instance.$SUBDOMAIN.$DOMAIN.yml
