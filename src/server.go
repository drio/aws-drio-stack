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
// TODO:
// Currently everything is hardcoded: /apps/test/foo/bar.html -> http://localhost:9000/foo/bar.html
// If we can't match the path, we respond with a 200 and a default body
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
	//rootURL, err := url.Parse(fmt.Sprintf("https://%s.%s", *env, *domain))
	rootURL, err := url.Parse(fmt.Sprintf("http://localhost:8080"))
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
	// TODO: check in the other server to make sure you have access to the SAML info
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
