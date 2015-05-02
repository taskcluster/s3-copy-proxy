package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

var httpClient = &http.Client{}

type Routes struct {
	config *ProxyConfig
}

func (self *Routes) constructSourceUrl(reqUrl *url.URL) url.URL {
	src := *self.config.Source
	src.Path = path.Join(src.Path, reqUrl.Path)
	return src
}

// Routes implements the `http.Handler` interface
func (self Routes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Copy method and body over to the proxy request.

	sourceURL := self.constructSourceUrl(req.URL)
	log.Printf("Proxying %s -> %s", req.URL, &sourceURL)

	proxyReq, err := http.NewRequest(req.Method, sourceURL.String(), req.Body)
	// If we fail to create a request notify the client.
	if err != nil {
		res.WriteHeader(500)
		fmt.Fprintf(res, "Failed to generate proxy request: %s", err)
		return
	}

	// Copy all headers over to the proxy request.
	for key, _ := range req.Header {
		// Do not forward connection!
		if key == "Connection" || key == "Host" {
			continue
		}

		proxyReq.Header.Set(key, req.Header.Get(key))
	}

	if err != nil {
		res.WriteHeader(500)
		fmt.Fprintf(res, "Failed to sign proxy request")
		return
	}

	// Issue the proxy request...
	proxyResp, err := httpClient.Do(proxyReq)

	if err != nil {
		res.WriteHeader(500)
		fmt.Fprintf(res, "Failed during proxy request: %s", err)
		return
	}

	// Map the headers from the proxy back into our proxyResponse
	headersToSend := res.Header()
	for key, _ := range proxyResp.Header {
		headersToSend.Set(key, proxyResp.Header.Get(key))
	}

	// Write the proxyResponse headers and status.
	res.WriteHeader(proxyResp.StatusCode)

	// Proxy the proxyResponse body from the endpoint to our response.
	io.Copy(res, proxyResp.Body)
	proxyResp.Body.Close()
}
