// vim: set foldmethod=indent foldlevel=1 et:
package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml/samlsp"
)

func root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
	<html lang="en">
	<meta charset=utf-8>
	<p>
		Welcome. You don't need to authenticate to see this. 🤘. Try <a href='/hello'>/hello</a> instead.
	</p>
	`)
}

func rootPlain(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Fprintf(w, "/ welcome. -- %s", hostname)
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
	keyPair, err := tls.LoadX509KeyPair("../cert/drioservice.cert", "../cert/drioservice.key")
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

	rootURL, err := url.Parse("http://staging.drtufts.net:80")
	if err != nil {
		panic(err) // TODO handle error
	}

	samlSP, _ := samlsp.New(samlsp.Options{
		URL:         *rootURL,
		Key:         keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate: keyPair.Leaf,
		IDPMetadata: idpMetadata,
	})

	app := http.HandlerFunc(hello)
	http.Handle("/hello", samlSP.RequireAccount(app))
	http.Handle("/saml/", samlSP)
	rootHandle := http.HandlerFunc(rootPlain)
	http.Handle("/", rootHandle)
	go (func() {
		fmt.Println("Listening HTTP:8080... ")
		http.ListenAndServe(":8080", nil)
	})()

	fmt.Println("Listening HTTPS:8443... ")
	err = http.ListenAndServeTLS(":8443", "../cert/server-cert.pem", "../cert/server-key.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}
