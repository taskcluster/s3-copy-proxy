package main

import (
	"fmt"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"log"
	"net/http"
	"net/url"
	"strconv"

	docopt "github.com/docopt/docopt-go"
)

func strToRegion(region string) (*aws.Region, error) {
	switch region {
	case "us-east-1":
		return &aws.USEast, nil
	case "us-west-1":
		return &aws.USWest, nil
	case "us-west-2":
		return &aws.USWest2, nil
	case "eu-west-1":
		return &aws.EUWest, nil
	case "eu-central-1":
		return &aws.EUCentral, nil
	case "ap-southeast-1":
		return &aws.APSoutheast, nil
	case "ap-southeast-2":
		return &aws.APSoutheast2, nil
	case "ap-northeast-1":
		return &aws.APNortheast, nil
	case "sa-east-1":
		return &aws.SAEast, nil
	case "cn-north-1":
		return &aws.CNNorth, nil
	}

	return nil, fmt.Errorf("Unknown region %s", region)
}

type ProxyConfig struct {
	Source *url.URL
	Bucket *s3.Bucket
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

	awsRegionObj, err := strToRegion(region)
	if err != nil {
		log.Fatal(err)
	}

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}

	client := s3.New(auth, *awsRegionObj)
	s3Bucket := client.Bucket(bucket)

	config := ProxyConfig{
		Source: url,
		Bucket: s3Bucket,
		Prefix: prefix,
	}

	log.Printf("Proxy server starting on port %d", port)

	routes := Routes{config: &config}
	startErr := http.ListenAndServe(fmt.Sprintf(":%d", port), routes)
	if startErr != nil {
		log.Fatal(startErr)
	}
}
