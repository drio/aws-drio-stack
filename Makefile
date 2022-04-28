# vim: set foldmethod=indent foldlevel=0:
ENV?=staging
DOMAIN?=drtufts.net
EC2_IP?=
REMOTE_DIR=/home/ec2-user/services
SERVICE_NAME=mainserver
REMOTE_SERVICE_DIR=$(REMOTE_DIR)/$(SERVICE_NAME)

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
	cd src; go run server.go -h

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

## ssh: ssh to instance
.PHONY: ssh
ssh:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP)

.PHONY: mkremotedir
mkremotedir:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "mkdir -p $(REMOTE_SERVICE_DIR)"

## rsync: rsync code to machine
.PHONY: rsync
rsync: mkremotedir
	rsync -avz \
		-e "ssh -i $(EC2_CER)" \
		--exclude=src/server \
		.  $(EC2_USER)@$(EC2_IP):$(REMOTE_SERVICE_DIR)

## rsync/bin: rsync bin (testing)
.PHONY: rsync/bin
rsync/bin:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "mkdir -p canonical_app"
	rsync -avz \
		-e "ssh -i $(EC2_CER)" \
		--exclude=src/server \
		/Users/drio/dev/github.com/drio/go-canonical-app/server.linux.amd64 \
		/Users/drio/dev/github.com/drio/go-canonical-app/sql/.env \
		/Users/drio/dev/github.com/drio/go-canonical-app/Makefile \
		/Users/drio/dev/github.com/drio/go-canonical-app/static \
		$(EC2_USER)@$(EC2_IP):canonical_app/

## rsync/all: deploy/do all
.PHONY: rsync/all
rsync/all:
	p=`pwd` && \
	cd /Users/drio/dev/github.com/drio/sk-canonical-app &&  rm -rf build && make build toserver &&  \
	cd ~/dev/github.com/drio/go-canonical-app && make build && \
	cd $$p && \
	make rsync rsync/bin EC2_IP=$$(./scripts/getips.sh staging 1) && \
	make rsync rsync/bin EC2_IP=$$(./scripts/getips.sh staging 2)

## metadata: run a curl request against the server to get the metadata
.PHONY: metadata
metadata:
	@curl -s $(URL)/saml/metadata

## run-test-server: run an http server for testing purposes
.PHONY: run-test-server
run-test-server:
	mkdir -p public
	#curl http://169.254.169.254/latest/meta-data/public-hostname > public/index.html
	echo "this is  the test server" > public/index.html
	cd public; python3 -m http.server 9000

## deploy: deploy new code and restart server
.PHONY: deploy
deploy: rsync
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "cd $(REMOTE_SERVICE_DIR) && make service/restart"

## remote/service/status: service status
.PHONY: remote/service/status
remote/service/status:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "systemctl status goserver"

## remote/service/%: install service on remote machine env=(prod, staging)
.PHONY: remote/service/install
remote/service/install:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "cd $(REMOTE_SERVICE_DIR) && sudo make service/install ENV=$(ENV)"

## remote/service/install: uninstall/remove service from remote machine
.PHONY: remote/service/uninstall
remote/service/uninstall:
	ssh -i $(EC2_CER) $(EC2_USER)@$(EC2_IP) "cd $(REMOTE_SERVICE_DIR) && sudo make service/uninstall"

## service/install: install the systemd service on current machine
.PHONY: service/install
service/install:
	cd $(REMOTE_SERVICE_DIR)/src && \
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
	cd $(REMOTE_SERVICE_DIR) && \
	rm -f src/server && \
	cd src && /usr/local/go/bin/go build server.go && \
	sudo systemctl start goserver.service

mod: go.mod

go.mod:
	go mod init github.com/drio/aws-drio-stack
