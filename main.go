package main

import (
	//"github.com/mitchellh/goamz/aws"
	//"github.com/mitchellh/goamz/s3"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	//"os"

	docopt "github.com/docopt/docopt-go"
)

type ProxyConfig struct {
	Source *url.URL
	Region string
	Bucket string
	Prefix string
}

var version = "s3-copy-proxy 1.0"
var usage = `

AWS Copy Proxy

This proxy is intended to reduce the amount of overall network traffic across
aws regions. Note that the intetion is to run at least one of these servers per
region.

Note it is expected that environment variables will be used to pass AWS credentails...

  Usage:
    proxy --source=<host> --region=<region> --bucket=<name> [--prefix=<path> --port=<port>]
    proxy --help

  Options:
    --source=<host>   Where to replicate content from.
    --region=<region> AWS Region where the bucket resides in.
    --bucket=<name>   Bucket Name.
    --prefix=<path>   Prefix to use within bucket when replicating. [deafult:]
    --port=<number>   Port to bind to [default: 8080]

  Examples:
    proxy --source=https://s3-us-west-2.amazonaws.com/taskcluster-public-artifacts \
      --region=us-east-1 \
      --bucket=taskcluster-public-artifacts-us-east-1 \
      --prefix=production
`

func main() {
	arguments, err := docopt.Parse(usage, nil, true, version, false, true)
	if err != nil {
		log.Fatal(err)
	}

	// Convert arguments into their appropriate go types...
	source := arguments["--source"].(string)
	region := arguments["--region"].(string)
	bucket := arguments["--bucket"].(string)

	port, err := strconv.Atoi(arguments["--port"].(string))
	if err != nil {
		log.Fatalf("Cannot parse port into int: %v", err)
	}

	var prefix string
	if arguments["--prefix"] == nil {
		prefix = ""
	} else {
		prefix = arguments["--prefix"].(string)
	}

	url, err := url.Parse(source)
	if err != nil {
		log.Fatalf("Error parsing source into url : %v", err)
	}

	config := ProxyConfig{
		Source: url,
		Region: region,
		Bucket: bucket,
		Prefix: prefix,
	}

	log.Printf("Proxy server starting on port %d", port)

	routes := Routes{config: &config}
	startErr := http.ListenAndServe(fmt.Sprintf(":%d", port), routes)
	if startErr != nil {
		log.Fatal(startErr)
	}
}
