SERVICE_NAME?=drioservice
DOMAIN?=example.com
MD_FILE?=saml-test-drio-localhost.xml

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run: start go server dev mode
.PHONY: run
run: cert/$(SERVICE_NAME).cert cert/$(SERVICE_NAME).key
	cd src; go run server.go

cert/$(SERVICE_NAME).cert cert/$(SERVICE_NAME).key: cert
	openssl req -x509 \
		-newkey rsa:2048 \
		-keyout cert/$(SERVICE_NAME).key \
	  -out cert/$(SERVICE_NAME).cert \
	  -days 365 \
	  -nodes \
	  -subj "/CN=$(SEVICE_NAME).$(DOMAIN)"

cert:
	mkdir cert

mod: go.mod

go.mod:
	go mod init github.com/drio/aws-drio-stack

metadata: $(MD_FILE)

$(MD_FILE):
	curl -s localhost:8000/saml/metadata > $@

clean:
	rm -f saml-test-drio-localhost.xml *.cert *.key
