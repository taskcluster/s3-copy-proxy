package main

import (
	"fmt"
	"github.com/goamz/goamz/s3"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

var httpClient = &http.Client{}

type Routes struct {
	config *ProxyConfig
}

func (self *Routes) constructKeyName(reqUrl *url.URL) string {
	key := path.Join(self.config.Prefix, reqUrl.Path)
	// Strip any starting slashes out of the bucket path...
	if key[0:1] == "/" {
		key = key[1:]
	}
	return key
}

func (self *Routes) constructSourceUrl(reqUrl *url.URL) url.URL {
	src := *self.config.Source
	src.Path = path.Join(src.Path, reqUrl.Path)
	return src
}

// Routes implements the `http.Handler` interface
func (self Routes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Check if we should directly redirect to s3 first...
	bucketKey := self.constructKeyName(req.URL)
	bucketKeyExists, err := self.config.Bucket.Exists(bucketKey)

	if err != nil {
		log.Print("Non fatal error checking if object is cached %v", err)
	}

	if bucketKeyExists {
		redirectUrl := self.config.Bucket.URL(bucketKey)
		log.Printf("Cache hit redirect %s", redirectUrl)
		http.Redirect(res, req, redirectUrl, 302)
		return
	} else {
		log.Printf("Cache miss %s", bucketKey)
	}

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

	// AWS will _not_ sent back content length if we don't set these in some
	// cases.
	if req.Header.Get("Accept-Encoding") == "" {
		proxyReq.Header.Set("Accept-Encoding", "gzip, deflate")
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
		log.Printf("Response header %s = %s", key, proxyResp.Header.Get(key))
		headersToSend.Set(key, proxyResp.Header.Get(key))
	}

	// Write the proxyResponse headers and status.
	res.WriteHeader(proxyResp.StatusCode)

	// If the proxy returns a successful status code replicate!
	if proxyResp.StatusCode == 200 {
		contentLengthInt, err := strconv.Atoi(proxyResp.Header.Get("Content-Length"))
		if err != nil {
			res.WriteHeader(500)
			fmt.Fprintf(res, "Invalid content length in source object...")
		}

		var contentLength int64
		contentLength = int64(contentLengthInt)

		// The magic is here.. This will allow the S3 upload to occur while
		// streaming the artifact back...
		reader := io.TeeReader(proxyResp.Body, res)
		self.config.Bucket.PutReader(
			bucketKey,
			reader,
			contentLength,
			proxyResp.Header.Get("Content-Type"),
			s3.PublicRead,
			s3.Options{},
		)
		log.Printf("%s", contentLength)
	} else {
		io.Copy(res, proxyResp.Body)
	}
}
