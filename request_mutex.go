package main

import (
	"fmt"
	"sync"
)

type requestMutex struct {
	sync.Mutex

	// The key is intended to be the path part of the request url...
	requests map[string]*chan bool
}

func newRequestMutex() requestMutex {
	requests := make(map[string]*chan bool)
	return requestMutex{
		requests: requests,
	}
}

func (self requestMutex) Get(name string) *chan bool {
	defer self.Unlock()
	self.Lock()

	return self.requests[name]
}

func (self requestMutex) Create(name string) (*chan bool, error) {
	defer self.Unlock()
	self.Lock()

	observer := make(chan bool)

	if self.requests[name] != nil {
		return nil, fmt.Errorf("Will not override existing request %s", name)
	}

	self.requests[name] = &observer
	return &observer, nil
}

// Complete and remove the request.
func (self requestMutex) Complete(name string, obj *chan bool) error {
	defer self.Unlock()
	self.Lock()

	if self.requests[name] == obj {
		close(*obj)
		delete(self.requests, name)
		return nil
	}
	return fmt.Errorf("Unknown request name %s", name)
}
