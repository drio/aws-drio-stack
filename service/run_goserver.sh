#!/bin/bash
set -e

cd /home/ec2-user/services/mainserver/src
rm -f goserver
/usr/local/go/bin/go build -o goserver

# TODO: currently hardcoded for staging
./goserver \
-idpurl="https://shib-idp-stage.uit.tufts.edu/idp/shibboleth" \
-rooturl="https://staging.drtufts.net"
