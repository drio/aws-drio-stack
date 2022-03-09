SERVICE_NAME?=drioservice
DOMAIN?=example.com
VERSION=2
MD_FILE?=saml-test-driov$(VERSION)-localhost.xml

run: $(SERVICE_NAME).cert $(SERVICE_NAME).key
	go run server.go

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

metadata: $(MD_FILE)

$(MD_FILE):
	curl -s localhost:8000/saml/metadata > $@

clean:
	rm -f saml-test-drio-localhost.xml *.cert *.key
