SERVICE_NAME?=drioservice
DOMAIN?=example.com
MD_FILE?=saml-test-drio-localhost.xml

URL=https://staging.drtufts.net
EC2_IP?=
EC2_USER?=ec2-user
EC2_CER?=~drio/.ssh/drio_aws_tufts.cer

HOST_DNS=ec2-18-223-239-5.us-east-2.compute.amazonaws.com
PORT=8080
MD_FILE=saml-test-drio-$(HOST_DNS).xml

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run: start go server dev mode
.PHONY: run
run:
	cd src; go run server.go

## cert: create x509 cert to interact with the IDP
.PHONY: cert
cert: 
	mkdir cert
	openssl req -x509 \
		-newkey rsa:2048 \ 
		-keyout cert/$(SERVICE_NAME).key \
	  -out cert/$(SERVICE_NAME).cert \
	  -days 365 \
	  -nodes \
	  -subj "/CN=$(SEVICE_NAME).$(DOMAIN)"

## server-cert: create x509 cert for the communication between the loadbalancer and the server
.PHONY: server-cert
server-cert:
	mkdir -p cert
	openssl req -new -newkey rsa:4096 -days 3650 \
	-nodes -x509 -subj "/C=/ST=/L=/O=/CN=localhost" \
	-keyout cert/server-key.pem \
	-out cert/server-cert.pem

## aws/list-ec2: list ec2 instances
.PHONY: aws/list-ec2
aws/list-ec2:
	aws ec2 describe-instances

## aws/ssh: ssh to instance
.PHONY: ssh
ssh:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP)

## rsync: rsync code to machine
.PHONY: rsync
rsync:
	rsync -avz -e "ssh -i $(EC2_CER)" . $(EC2_USER)@$(EC2_IP):

mod: go.mod

go.mod:
	go mod init github.com/drio/aws-drio-stack

metadata:
	@curl -s $(URL)/saml/metadata

## run-test-server: run an http server for testing purposes
.PHONY: run-test-server
run-test-server:
	mkdir -p public
	#curl http://169.254.169.254/latest/meta-data/public-hostname > public/index.html
	cat /etc/hostname > public/index.html
	cd public; python -m SimpleHTTPServer $(PORT)
