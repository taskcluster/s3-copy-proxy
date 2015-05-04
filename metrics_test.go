package main

import (
	influxdb "github.com/influxdb/influxdb/client"
	"testing"
)

// This is a fairly lame test which basically ensures that this does not crash
// under some conditions... While the logic is fairly simple there is zero
// testing of how it actually works when submitting to influxdb (which would
// require some additional external deps)
func TestMetrics(t *testing.T) {
	metrics, err := NewMetrics()
	if err != nil {
		t.Fatal(err)
	}
	metrics.Send(&influxdb.Series{
		Name: "testing",
		Columns: []string{
			"foo",
		},
		Points: [][]interface{}{
			{
				"bar",
			},
		},
	})
}
