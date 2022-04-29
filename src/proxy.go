package main

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/crewjam/saml/samlsp"
)

const (
	redirectHostURL string = "http://127.0.0.1"
	pathAppsName    string = "apps"
)

type proxyMatch struct {
	appName, targetPort string
}

// appDirNames: "canonical"
// targetPort: port to proxy to in localhost
func proxyRequest(w http.ResponseWriter, r *http.Request, match proxyMatch) bool {
	log.Printf("proxyRequest(), r.URL.Path=%s\n", r.URL.Path)
	sPath := strings.Split(r.URL.Path, "/")

	if len(sPath) > 2 && sPath[1] == pathAppsName && sPath[2] == match.appName {
		// Set the proper content-type
		ext := filepath.Ext(r.URL.Path)
		mimeString := mime.TypeByExtension(ext)
		if len(mimeString) > 0 {
			w.Header().Set("Content-Type", mimeString)
			log.Printf("proxyRequest(): ext='%s' mimeString='%s'\n", ext, mimeString)
		}

		// Before proxing, change the path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/%s/%s", pathAppsName, match.appName))
		targetUrl := fmt.Sprintf("%s:%s", redirectHostURL, match.targetPort)
		remote, err := url.Parse(targetUrl)
		if err != nil {
			log.Printf("proxyRequest(): ERROR: %s", err)
			return false
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
		return true
	}
	return false
}

// Creates the handler
// If the URL's path is of the form /apps/<app>/... then we use the <app> part of the Path
// to proxy the request to the proper service/server.
func genHandler(env string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		uid := samlsp.AttributeFromContext(r.Context(), "uid")
		r.Header.Add("Sb-Uid", uid)

		gotProxied := false
		matches := []proxyMatch{{"canonical", "9000"}, {"test", "9001"}}
		for _, pm := range matches {
			gotProxied = proxyRequest(w, r, pm)
			if gotProxied {
				break
			}
		}
		if !gotProxied {
			fmt.Fprintf(w, "👋 / welcome v3!. [%s] , env:%s -- %s", r.URL.Path, env, hostname)
		}
	}
}
