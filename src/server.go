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
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

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

// Creates the handler
// If the URL's path is of the form /apps/<app>/... then we use the <app> part of the Path
// to proxy the request to the proper service/server.
func genHandler(env string) func(http.ResponseWriter, *http.Request) {
	target := "http://localhost:9000"
	remote, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)

	return func(w http.ResponseWriter, r *http.Request) {
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		sPath := strings.Split(r.URL.Path, "/")
		if len(sPath) > 2 && sPath[1] == "apps" && sPath[2] == "test" {
			r.URL.Path = strings.Replace(r.URL.Path, "/apps/test", "", 1)
			// TODO: Add the SAML attributes you want before proxing
			proxy.ServeHTTP(w, r)
		} else {
			fmt.Fprintf(w, "/ welcome v2. [%s] , env:%s -- %s", r.URL.Path, env, hostname)
		}
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

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/"
		p.ServeHTTP(w, r)
	}
}

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
	})

	app := http.HandlerFunc(hello)
	slo := http.HandlerFunc(logout)
	http.Handle("/hello", samlMiddleware.RequireAccount(app))
	http.Handle("/saml/", samlMiddleware)
	http.Handle("/logout", slo)

	//rootHandle := http.HandlerFunc(genHandler(*env))
	//http.Handle("/", samlMiddleware.RequireAccount(rootHandle))
	rootHandle := http.HandlerFunc(hello)
	http.Handle("/", rootHandle)

	go (func() {
		log.Println("Listening HTTP:8080... ")
		http.ListenAndServe("127.0.0.1:8080", nil)
	})()

	log.Println("Listening HTTPS:8443... ")
	err = http.ListenAndServeTLS("127.0.0.1:8443", "../cert/server-cert.pem", "../cert/server-key.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}
