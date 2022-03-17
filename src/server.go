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

func logout(w http.ResponseWriter, r *http.Request) {
	nameID := samlsp.AttributeFromContext(r.Context(), "urn:oasis:names:tc:SAML:attribute:subject-id")
	url, err := samlMiddleware.ServiceProvider.MakeRedirectLogoutRequest(nameID, "")
	if err != nil {
		panic(err) // TODO handle error
	}

	err = samlMiddleware.Session.DeleteSession(w, r)
	if err != nil {
		panic(err) // TODO handle error
	}

	w.Header().Add("Location", url.String())
	w.WriteHeader(http.StatusFound)
}

func root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
	<html lang="en"> <meta charset=utf-8>
	<p>
		Welcome. You don't need to authenticate to see this. 🤘. Try <a href='/hello'>/hello</a> instead.
	</p>
	`)
}

func genHandler(env string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Fprintf(w, "/ welcome v2. env:%s -- %s", env, hostname)
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Fprintf(w, "- Hello, %s (%s)!", samlsp.AttributeFromContext(r.Context(), "uid"), hostname)
}

func main() {
	env := flag.String("env", "staging", "Environment: staging or prod")
	domain := flag.String("domain", "", "Domain name")
	flag.Parse()
	if *env != "staging" && *env != "prod" {
		log.Println("Only valid environments are: staging or prod")
		os.Exit(0)
	}
	if *domain == "" {
		log.Println("Invalid domain name: ", *domain)
		os.Exit(0)
	}

	keyPair, err := tls.LoadX509KeyPair("../cert/saml.cert", "../cert/saml.key")
	if err != nil {
		panic(err) // TODO handle error
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		panic(err) // TODO handle error
	}

	idpMetadataURL, err := url.Parse("https://samltest.id/saml/idp")
	if err != nil {
		panic(err) // TODO handle error
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient,
		*idpMetadataURL)
	if err != nil {
		panic(err) // TODO handle error
	}

	//"https://staging.drtufts.net"
	rootURL, err := url.Parse(fmt.Sprintf("https://%s.%s", *env, *domain))
	if err != nil {
		panic(err) // TODO handle error
	}

	samlMiddleware, _ = samlsp.New(samlsp.Options{
		URL:         *rootURL,
		Key:         keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate: keyPair.Leaf,
		IDPMetadata: idpMetadata,
		//SignRequest: true, // some IdP require the SLO request to be signed
	})

	log.Println("Domain: ", *domain)
	log.Println("Env: ", *env)

	app := http.HandlerFunc(hello)
	slo := http.HandlerFunc(logout)
	http.Handle("/hello", samlMiddleware.RequireAccount(app))
	http.Handle("/saml/", samlMiddleware)
	http.Handle("/logout", slo)

	rootHandle := http.HandlerFunc(genHandler(*env))
	http.Handle("/", rootHandle)
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
