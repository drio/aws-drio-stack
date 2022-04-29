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

func logout(w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse("https://staging.drtufts.net/bye")
	if err != nil {
		panic(err)
	}

	err = samlMiddleware.Session.DeleteSession(w, r)
	if err != nil {
		panic(err) // TODO handle error
	}

	w.Header().Add("Location", url.String())
	w.WriteHeader(http.StatusFound)
}

func bye(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
  <style>
    :root {
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
        Segoe UI, Roboto, Helvetica Neue, Arial, Noto Sans, sans-serif,
        "Apple Color Emoji", "Segoe UI Emoji", Segoe UI Symbol, "Noto Color Emoji";
      margin: 0 auto;
      width: 75%%;
      background: white;
      padding: 10px;
    }

    .container {
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    img {
      width: 50%%;
    }
  </style>
  <div class="container">
    <p>Sorry to see you go. Bye now.</p>
    <img src="https://i.giphy.com/media/k3YNVBrbn2KqbXEgDJ/giphy.webp"/>
  </div>
  `)
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

func rootPage(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
  <style>
    :root {
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
        Segoe UI, Roboto, Helvetica Neue, Arial, Noto Sans, sans-serif,
        "Apple Color Emoji", "Segoe UI Emoji", Segoe UI Symbol, "Noto Color Emoji";
      margin: 0 auto;
      width: 75%%;
      background: white;
      padding: 10px;
    }

    .container {
      border-left: solid 5px #ffdcef;
      padding-left: 20px;
    }

    .served {
      font-size: .8rem;
      color: grey;
    }

    a {
      text-decoration: none;
      color: black;
    }

    .button {
      width: fit-content;
      border: solid 2px steelblue;
      padding: 5px;
      border-radius: 5px;
    }

    .uid {
      color: tomato;
      font-weight: 500;
     }
  </style>
  <div class="container">
    <p>🎉 Congratulations <span class="uid">%s</span>, you are authenticated. </p>
    <p class="served"> Served from: (%s)</p>
    <p>List of available apps:</p>
    <ul>
      <li><a href="/apps/test">test</a></li>
      <li><a href="/apps/canonical">canonical</a></li>
    </ul>
    <p class="button"><a href="/logout">Logout</a></p>
  </div>
  `, samlsp.AttributeFromContext(r.Context(), "uid"), hostname)
}

// Creates the handler
// If the URL's path is of the form /apps/<app>/... then we use the <app> part of the Path
// to proxy the request to the proper service/server.
func genRootHandler(env string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
			rootPage(w, r)
		}
	}
}
