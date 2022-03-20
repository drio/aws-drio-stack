// This is a little testing app server. We can use it to test the main proxy
// server to make sure it is proxing request properly
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/crewjam/saml/samlsp"
)

func root(w http.ResponseWriter, r *http.Request) {
	subjectID := samlsp.AttributeFromContext(r.Context(), "urn:oasis:names:tc:SAML:attribute:subject-id")
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en"> <meta charset=utf-8>
<p>
	Welcome to this amazing app 🤘. <br/>
	user-agent: %s
	subjectID: %s
</p>
`, r.Header["User-Agent"], subjectID)
}

func headers(w http.ResponseWriter, req *http.Request) {
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func main() {
	port := flag.String("port", "", "Port to listen to")
	if *port == "" {
		*port = "9000"
	}
	http.HandleFunc("/", root)
	http.HandleFunc("/headers", headers)
	fmt.Printf("Listening on port %s\n", *port)
	err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%s", *port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
