package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

// Note: We could use goamz or another aws specific lookup but it's easier to
// do it ourselves which also has the benefit of allowing easier testing.
const AWS_METADATA_URL = "http://169.254.169.254/latest/meta-data/"

var metadataHttpClient = http.Client{
	// This should always be very fast...
	Timeout: 10 * time.Second,
}

// Shortcut helper for fetching metadata..
func getMetadata(base string, key string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	// Safely join the base path with the key...
	baseURL.Path = path.Join(baseURL.Path, key)

	resp, err := metadataHttpClient.Get(baseURL.String())
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Unknown metadata key %s (%d)", resp.StatusCode)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

type HostType interface {
	Details() (*HostDetails, error)
	Description() string
}

type UnknownHostType struct{}

func (self UnknownHostType) Description() string {
	return "Unknown host (non aws)"
}

func (self UnknownHostType) Details() (*HostDetails, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	result := &HostDetails{
		Hostname:     hostname,
		Region:       "?",
		InstanceID:   "?",
		InstanceType: "?",
	}
	return result, nil
}

type AWSHostType struct {
	BaseURL string
}

func (self AWSHostType) Description() string {
	return "AWS Host"
}

func (self AWSHostType) Details() (*HostDetails, error) {
	// It's lazy to fetch them sequentially but it does provide better error
	// handling...

	hostname, err := getMetadata(self.BaseURL, "public-hostname")
	if err != nil {
		return nil, err
	}

	region, err := getMetadata(self.BaseURL, "placement/availability-zone")
	if err != nil {
		return nil, err
	}

	instanceType, err := getMetadata(self.BaseURL, "instance-type")
	if err != nil {
		return nil, err
	}

	instanceID, err := getMetadata(self.BaseURL, "instance-id")
	if err != nil {
		return nil, err
	}

	result := &HostDetails{
		Hostname:     hostname,
		Region:       region,
		InstanceType: instanceType,
		InstanceID:   instanceID,
	}

	return result, nil
}

type HostDetails struct {
	Hostname     string
	Region       string
	InstanceType string
	InstanceID   string
}

func GetHostType(metadataURL string) HostType {
	// Attempt to determine the host detail type.
	if metadataURL == "" {
		metadataURL = AWS_METADATA_URL
	}

	_, metaFetchErr := getMetadata(metadataURL, "")
	if metaFetchErr != nil {
		return UnknownHostType{}
	}
	return AWSHostType{
		BaseURL: metadataURL,
	}
}
