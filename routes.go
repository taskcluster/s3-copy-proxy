package main

import (
	//"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
)

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
	log.Printf("Proxying %s | %s | %s", req.URL, req.Method, self.constructSourceUrl(req.URL))
	//proxyReq, err := http.NewRequest(req.Method, targetPath.String(), req.Body)
}
