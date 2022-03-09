SERVICE_NAME?=drioservice
DOMAIN?=example.com

$(SERVICE_NAME).cert $(SERVICE_NAME).key:
	openssl req -x509 \
		-newkey rsa:2048 \
		-keyout $(SERVICE_NAME).key \
	  -out $(SERVICE_NAME).cert \
	  -days 365 \
	  -nodes \
	  -subj "/CN=$(SEVICE_NAME).$(DOMAIN)"

mod: go.mod

go.mod:
	go mod init github.com/drio/aws-drio-stack
