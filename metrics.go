package main

import (
	"fmt"
	influxdb "github.com/influxdb/influxdb/client"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const SEND_INTERVAL = 30 * time.Second

// Metrics is thread safe and non-blocking (as far as the influx bit goes)
type Metrics struct {
	sync.Mutex
	// True when we have credentials and actually send data to influxdb
	Active        bool
	pendingWrites []*influxdb.Series

	// Client responsible for sending metrics _may_ be null if Active is false
	client *influxdb.Client

	// This always starts as nil (Start should set this value)
	ticker *time.Ticker
}

func (self *Metrics) Send(series *influxdb.Series) {
	if !self.Active {
		// If we are not actively able to send metrics this is a lock-less no-op
		return
	}

	defer self.Unlock()
	self.Lock()

	self.pendingWrites = append(self.pendingWrites, series)
}

func (self *Metrics) Start() error {
	// Safe guard callers which may call this without an active metrics instance.
	if !self.Active {
		return fmt.Errorf("Metrics will not send events without valid credentials...")
	}

	if self.ticker != nil {
		return fmt.Errorf("Cannot be started more then once!")
	}

	self.ticker = time.NewTicker(SEND_INTERVAL)
	log.Printf("Starting periodic sender will send every %s", SEND_INTERVAL)
	go self.periodicSender(self.ticker)
	return nil
}

// This function will loop forever sending metrics.
func (self *Metrics) periodicSender(ticker *time.Ticker) {
	tickChan := ticker.C
	for {
		select {
		case <-tickChan:
			// Fire and forget (though we will log any errors...)
			go func() {
				err := self.SendMetrics()
				if err != nil {
					log.Printf("Non fatal error while sending metrics", err)
				}
			}()
		}
	}
}

// Immediately send metrics... This is intended to be called by the ticker but
// may also be called directly if you need to synchronously send metrics for
// some reason...
func (self *Metrics) SendMetrics() error {
	// We don't need to be locked for the entire duration if this method only when
	// we mutate the pending writes.
	self.Lock()
	pendingWrites := self.pendingWrites
	self.pendingWrites = []*influxdb.Series{}
	self.Unlock()

	if len(pendingWrites) > 0 {
		log.Printf("Sending %d metrics", len(pendingWrites))
		return self.client.WriteSeries(pendingWrites)
	}
	return nil
}

// Constructs the metrics sender... This will always succeed but only sends
// metrics if the ININFLUXDB_URL environment variable is set! If the value is
// set this will log warning (but not panic) if there are errors sending the
// metrics.
func NewMetrics() (*Metrics, error) {
	pendingWrites := []*influxdb.Series{}

	connectionString := os.Getenv("INFLUXDB_URL")
	if connectionString == "" {
		log.Printf("INFLUXDB_URL environment variable empty no metrics will be sent")
		result := &Metrics{
			Active:        false,
			pendingWrites: pendingWrites,
		}
		return result, nil
	}

	connectionURL, err := url.Parse(connectionString)
	if err != nil {
		return nil, err
	}

	isSecure := false
	if strings.Contains(connectionURL.Scheme, "https") {
		isSecure = true
	}

	database := connectionURL.Path
	if database[0:1] == "/" {
		database = database[1:]
	}

	password, _ := connectionURL.User.Password()

	influxConfig := &influxdb.ClientConfig{
		Host:     connectionURL.Host,
		Username: connectionURL.User.Username(),
		Password: password,
		Database: database,
		IsSecure: isSecure,
	}

	influxClient, err := influxdb.NewClient(influxConfig)
	if err != nil {
		return nil, err
	}

	log.Printf("Successfully configured influxdb")
	log.Printf(
		"host=%s user=%s database=%s",
		influxConfig.Host,
		influxConfig.Username,
		influxConfig.Database,
	)

	result := &Metrics{
		Active:        true,
		client:        influxClient,
		pendingWrites: pendingWrites,
	}

	// Start metrics writer...
	err = result.Start()
	if err != nil {
		return nil, err
	}

	return result, nil
}
