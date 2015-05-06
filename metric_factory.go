package main

import (
	influxdb "github.com/influxdb/influxdb/client"
	"time"
)

const (
	CACHE_HIT_SERIES             = "CacheHit"
	CACHE_WAITED_FOR_UPLOAD      = "CacheWaitedForUpload"
	CACHE_WAITED_FOR_UPLOAD_MISS = "CacheWaitedForUploadMiss"
	CACHE_UPLOAD                 = "CacheUpload"
	CACHE_UPLOAD_ERR             = "CacheUploadError"
	CACHE_TIMEOUT                = "CacheTimeout"
	CACHE_ERR_REDIRECT           = "CacheErrorRedirect"
)

type MetricFactory struct {
	hostDetails *HostDetails
	proxyConfig *ProxyConfig
}

func NewMetricFactory(hostDetails *HostDetails, proxyConfig *ProxyConfig) MetricFactory {
	return MetricFactory{
		hostDetails: hostDetails,
		proxyConfig: proxyConfig,
	}
}

func (self *MetricFactory) CacheErrorRedirect() *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_ERR_REDIRECT,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
			},
		},
	}
}

func (self *MetricFactory) CacheTimeout(waitDuration time.Duration) *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_TIMEOUT,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
			"waited",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
				waitDuration.Seconds(),
			},
		},
	}
}

func (self *MetricFactory) CacheUpload(uploadDuration time.Duration, contentLength int64) *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_UPLOAD,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
			"uploadDuration",
			"contentLength",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
				uploadDuration.Seconds(),
				contentLength,
			},
		},
	}
}

func (self *MetricFactory) CacheUploadError(uploadDuration time.Duration, path string, contentLength int64, err error) *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_UPLOAD_ERR,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
			"uploadDuration",
			"contentLength",
			"path",
			"error",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
				uploadDuration.Seconds(),
				contentLength,
				path,
				err.Error(),
			},
		},
	}
}

func (self *MetricFactory) WaitedForUpload(time time.Duration) *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_WAITED_FOR_UPLOAD,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
			"waitTime",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
				time.Seconds(),
			},
		},
	}
}

func (self *MetricFactory) WaitedForUploadMiss(time time.Duration) *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_WAITED_FOR_UPLOAD_MISS,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
			"time",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
				time.Seconds(),
			},
		},
	}
}

func (self *MetricFactory) CacheHit() *influxdb.Series {
	return &influxdb.Series{
		Name: CACHE_HIT_SERIES,
		Columns: []string{
			"hostname",
			"region",
			"instanceType",
			"instanceID",
		},
		Points: [][]interface{}{
			{
				self.hostDetails.Hostname,
				self.hostDetails.Region,
				self.hostDetails.InstanceType,
				self.hostDetails.InstanceID,
			},
		},
	}
}
