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
	"time"
)

var httpClient = &http.Client{}

const MAX_SOURCE_PULL_WAIT = 90 * time.Second
const MAX_WAIT_HEADER = "x-max-wait-duration"

type Routes struct {
	config         *ProxyConfig
	requests       requestMutex
	metrics        *Metrics
	metricsFactory *MetricFactory
}

func NewRoutes(config *ProxyConfig, metrics *Metrics, metricsFactory *MetricFactory) Routes {
	return Routes{
		config:         config,
		requests:       newRequestMutex(),
		metrics:        metrics,
		metricsFactory: metricsFactory,
	}
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

func (self *Routes) redirectToSource(res http.ResponseWriter, req *http.Request) {
	source := self.constructSourceUrl(req.URL)
	http.Redirect(res, req, source.String(), 302)
}

// Attempt to redirect the given request to the cache bucket.
func (self *Routes) attemptCacheRedirect(key string, res http.ResponseWriter, req *http.Request) bool {
	bucketKeyExists, err := self.config.Bucket.Exists(key)

	if err != nil {
		log.Print("Non fatal error checking if object is cached %v", err)
	}

	if bucketKeyExists {
		redirectUrl := self.config.Bucket.URL(key)
		log.Printf("Cache hit redirect %s", redirectUrl)
		http.Redirect(res, req, redirectUrl, 302)
		return true
	}

	return false
}

func (self *Routes) cacheAndServeFromSource(
	key string,
	lock *chan bool,
	res http.ResponseWriter,
	req *http.Request,
) {
	// When we complete serving this free the lock...
	defer self.requests.Complete(key, lock)
	uploadStartTime := time.Now()

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

	// If the proxy returns a successful status code replicate!
	if proxyResp.StatusCode == 200 {
		contentLengthInt, err := strconv.Atoi(proxyResp.Header.Get("Content-Length"))
		if err != nil {
			res.WriteHeader(500)
			fmt.Fprintf(res, "Invalid content length in source object...")
		}

		res.WriteHeader(proxyResp.StatusCode)

		var contentLength int64
		contentLength = int64(contentLengthInt)

		// The magic is here.. This will allow the S3 upload to occur while
		// streaming the artifact back...
		reader := io.TeeReader(proxyResp.Body, res)
		err = self.config.Bucket.PutReader(
			key,
			reader,
			contentLength,
			proxyResp.Header.Get("Content-Type"),
			s3.PublicRead,
			s3.Options{},
		)

		if err != nil {
			self.metrics.Send(self.metricsFactory.CacheUploadError(
				time.Now().Sub(uploadStartTime),
				contentLength,
				err,
			))
		} else {
			self.metrics.Send(self.metricsFactory.CacheUpload(
				time.Now().Sub(uploadStartTime),
				contentLength,
			))
		}

	} else {
		// We don't care about any of the contents here if we can't cache them so
		// just redirect the user directly to the source...
		proxyResp.Body.Close()
		self.redirectToSource(res, req)
	}
}

// Wait for another request to complete the pull/cache or timeout and redirect
// to the source...
func (self *Routes) waitForSourcePull(
	key string,
	lock *chan bool,
	res http.ResponseWriter,
	req *http.Request,
) {

	now := time.Now()
	wait := MAX_SOURCE_PULL_WAIT

	// Primarily for testing we allow setting how long this request should wait
	// (it cannot be configured to wait for more then the default value though!)
	configuredWait := req.Header.Get(MAX_WAIT_HEADER)
	if configuredWait != "" {
		configuredWaitDuration, err := time.ParseDuration(configuredWait)
		if err != nil {
			log.Printf("Could not use configured wait (%s) %v", configuredWait, err)
		} else {
			if configuredWaitDuration <= wait {
				wait = configuredWaitDuration
			} else {
				log.Printf(
					"Configured max wait %d cannot exceed default of %d",
					configuredWaitDuration,
					wait,
				)
			}
		}
	}

	select {
	case <-*lock:
		waited := time.Now().Sub(now)
		log.Printf("%s ready waited for %v", key, waited)
		redirected := self.attemptCacheRedirect(key, res, req)
		if !redirected {
			self.metrics.Send(self.metricsFactory.WaitedForUploadMiss(waited))
			log.Printf("Successfully watied for %s but no cache was created", key)
			self.redirectToSource(res, req)
		} else {
			self.metrics.Send(self.metricsFactory.WaitedForUpload(waited))
		}
	case <-time.After(wait):
		waited := time.Now().Sub(now)
		log.Printf("Timed out while waiting for upload of %s", key)
		self.metrics.Send(self.metricsFactory.CacheTimeout(waited))
		self.redirectToSource(res, req)
	}
}

// Routes implements the `http.Handler` interface
func (self Routes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Check if we should directly redirect to s3 first...
	key := self.constructKeyName(req.URL)

	// Attempt the initial cache hit...
	if self.attemptCacheRedirect(key, res, req) {
		self.metrics.Send(self.metricsFactory.CacheHit())
		return
	}

	// Mutex around who can do the source pulling and when...
	lock := self.requests.Get(key)
	if lock != nil {
		log.Printf("Already pulling %s waiting...", key)
		self.waitForSourcePull(key, lock, res, req)
		return
	}

	// We could not do the simple cache hit case so see if we need to pull from
	// the source...
	lock, err := self.requests.Create(key)
	if err != nil {
		// The intention here is to do whatever it takes to serve the content even
		// if we can't do it optimally so we just log the error and redirect to the
		// source.
		log.Printf("Error getting lock to pull source artifact %v", err)
		self.redirectToSource(res, req)
		self.metrics.Send(self.metricsFactory.CacheErrorRedirect())
		return
	}

	// Pull from the source !
	self.cacheAndServeFromSource(key, lock, res, req)
}
