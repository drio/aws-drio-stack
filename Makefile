ENV?=staging
DOMAIN?=drtufts.net
EC2_IP?=
SERVICE_NAME=drioservice

URL=https://$(ENV).$(DOMAIN)
EC2_USER?=ec2-user
EC2_CER?=~drio/.ssh/drio_aws_tufts.cer

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run: start go server dev mode
.PHONY: run
run:
	cd src; go run server.go -domain=$(DOMAIN) -env=$(ENV)

## cert: create x509 cert to interact with the IDP
.PHONY: cert
cert: 
	mkdir cert
	openssl req -x509 \
		-newkey rsa:2048 \ 
		-keyout cert/saml.key \
	  -out cert/saml.cert \
	  -days 365 \
	  -nodes \
	  -subj "/CN=$(SERVICE_NAME).$(DOMAIN)"

## server-cert: create x509cert for communication between loadbalancer-server
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
	rsync -avz -e "ssh -i $(EC2_CER)" --exclude=src/server . $(EC2_USER)@$(EC2_IP):

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
	cd public; python -m SimpleHTTPServer 8080

## deploy: deploy new code and restart server
.PHONY: deploy
deploy: rsync
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "make service/restart"

## deploy/service/status: service status
.PHONY: deploy/service/status
deploy/service/status:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "systemctl status goserver"

## deploy/service/%: install service on remote machine env=(prod, staging)
.PHONY: deploy/service/install
deploy/service/install:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "sudo make service/install ENV=$(ENV)"

## deploy/service/install: uninstall/remove service from remote machine
.PHONY: deploy/service/uninstall
deploy/service/uninstall:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "sudo make service/uninstall"

## service/install: install the systemd service on current machine
.PHONY: service/install
service/install:
	cd src; \
	/usr/local/go/bin/go build server.go && \
	cd .. && \
	cat ./service/goserver.service | \
		sed 's/__ENV__/$(ENV)/g' | \
		sed 's/__DOMAIN__/$(DOMAIN)/g' \
		> /lib/systemd/system/goserver.service && \
	chmod 644 /lib/systemd/system/goserver.service && \
	systemctl daemon-reload && \
	systemctl enable goserver && \
	systemctl restart goserver

## service/uninstall: uninstall the systemd service on current machine
.PHONY: service/uninstall
service/uninstall:
	sudo systemctl stop goserver
	sudo systemctl disable goserver.service
	sudo rm -rf /etc/systemd/system/goserver.service /etc/systemd/user/goserver.service

## service/restart: restart service
.PHONY: service/restart
service/restart:
	sudo systemctl stop goserver.service  && \
	rm -f src/server && \
	cd src; /usr/local/go/bin/go build server.go && \
	sudo systemctl start goserver.service
