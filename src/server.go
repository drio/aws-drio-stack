// vim: set foldmethod=indent foldlevel=1 et:
package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml/samlsp"
)

var samlMiddleware *samlsp.Middleware

func printHelp(msg string) {
	if msg != "" {
		fmt.Println(fmt.Sprintf("ERROR: %s", msg))
	}
	fmt.Println(`Usage:
  $ ./server -env=<staging or prod> -idpurl=<idp metadata url> -rooturl=<server root url>

Examples:
  $ ./server -env=staging -idpurl="https://samltest.id/saml/idp" -rooturl="https://staging.drtufts.net"
  $ ./server -env=staging -idpurl="https://shib-idp-stage.uit.tufts.edu/idp/shibboleth" -rooturl="https://staging.drtufts.net"
  $ ./server -env=prod -idpurl="https://shib-idp.tufts.edu/idp/shibboleth" -rooturl="https://prod.drtufts.net"
`)
	os.Exit(0)
}

func main() {
	help := flag.Bool("h", false, "help")
	env := flag.String("env", "staging", "Environment: [staging or prod]")
	flagIdpMetadataURL := flag.String("idpurl", "", "idp metadata url")
	flagRootURL := flag.String("rooturl", "", "service provider main url")
	flag.Parse()

	if *help {
		printHelp("")
	}

	if *env != "staging" && *env != "prod" {
		printHelp("Only valid environments are: staging or prod")
	}

	if *flagIdpMetadataURL == "" {
		printHelp("No idpMetadataURL provided")
	}

	if *flagRootURL == "" {
		printHelp("No rootURL provided")
	}

	log.Printf("Env      [%s]\n", *env)
	log.Printf("IDP url  [%s]\n", *flagIdpMetadataURL)
	log.Printf("root url [%s]\n", *flagRootURL)

	keyPair, err := tls.LoadX509KeyPair("../cert/saml.cert", "../cert/saml.key")
	if err != nil {
		panic(err)
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		panic(err)
	}

	rootURL, err := url.Parse(*flagRootURL)
	if err != nil {
		panic(err)
	}

	idpMetadataURL, err := url.Parse(*flagIdpMetadataURL)
	if err != nil {
		panic(fmt.Sprintf("Error processing idp metatada url: %v\n", err))
	}

	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient,
		*idpMetadataURL)
	if err != nil {
		panic(fmt.Sprintf("Error fetching idp smetatada: %v\n", err))
	}

	samlMiddleware, _ = samlsp.New(samlsp.Options{
		URL:         *rootURL,
		Key:         keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate: keyPair.Leaf,
		IDPMetadata: idpMetadata,
		//SignRequest: true, // some IdP require the SLO request to be signed
		ForceAuthn: true,
	})

	//app := http.HandlerFunc(hello)
	//http.Handle("/hello", samlMiddleware.RequireAccount(app))
	/* handles the twilio callbacks */
	http.Handle("/callback", http.HandlerFunc(twilioCallbackHandler))
	http.Handle("/saml/", samlMiddleware)
	http.Handle("/logout", http.HandlerFunc(logout))
	http.Handle("/bye", http.HandlerFunc(bye))
	rootHandle := http.HandlerFunc(genRootHandler(*env))
	http.Handle("/", samlMiddleware.RequireAccount(rootHandle))

	go (func() {
		log.Println("Listening HTTP:8080... ")
		http.ListenAndServe(":8080", nil)
	})()

	log.Println("Listening HTTPS:8443... ")
	err = http.ListenAndServeTLS(":8443", "../cert/server-cert.pem", "../cert/server-key.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}
