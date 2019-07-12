package utils

import (
	"sync"
)

// ConcurrentMap - thread safe map
type ConcurrentMap struct {
	mx sync.RWMutex
	m  map[string]string
}

// Get - safely get value
func (c *ConcurrentMap) Get(key string) (string, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	val, ok := c.m[key]
	return val, ok
}

// Set - safely set value
func (c *ConcurrentMap) Set(key string, value string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.m[key] = value
}

// ToMap - returns inner map
func (c *ConcurrentMap) ToMap() map[string]string {
	return c.m
}

// NewConcurrentMap - create new thread safe map
func NewConcurrentMap() *ConcurrentMap {
	return &ConcurrentMap{
		m: make(map[string]string),
	}
}
