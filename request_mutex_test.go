package main

import (
	"sync"
	"testing"
)

func TestMultipleConsumers(t *testing.T) {
	key := "xfoobar/"
	requests := newRequestMutex()

	requestDone := requests.Get(key)
	if requestDone != nil {
		t.Fatalf("Request has key %s before starting", key)
	}

	requestDone, err := requests.Create(key)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			<-*requestDone
			wg.Done()
		}()
	}

	requests.Complete(key, requestDone)
	wg.Wait()
}
