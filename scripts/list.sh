#!/bin/bash -e

aws cloudformation list-exports \
  --query "Exports[?starts_with(Name,'InstanceEndpoint')].Value"
