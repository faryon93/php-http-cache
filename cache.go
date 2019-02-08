package main

// php-http-cache
// Copyright (C) 2019 Maximilian Pachl

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// ---------------------------------------------------------------------------------------
//  imports
// ---------------------------------------------------------------------------------------

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------------------
//  types
// ---------------------------------------------------------------------------------------

type CacheService struct {
	cache map[uint64]*CacheEntry
	mutex sync.RWMutex
}

type CacheEntry struct {
	Method  string
	Url     string
	Body    string
	Headers http.Header
	Ttl     time.Duration

	Fetching sync.RWMutex
	Response string
	Error    error
	Hash     string
}

// ---------------------------------------------------------------------------------------
//  public functions
// ---------------------------------------------------------------------------------------

func NewCacheService() *CacheService {
	return &CacheService{
		cache: make(map[uint64]*CacheEntry),
	}
}

// ---------------------------------------------------------------------------------------
//  public members
// ---------------------------------------------------------------------------------------

func (c *CacheService) Request(body string, r *string) error {
	// parse the request
	var request CmdRequest
	err := json.Unmarshal([]byte(body), &request)
	if err != nil {
		return err
	}
	id := request.Hash()

	// check if there is already a cache entry present -> if not create a new one
	c.mutex.Lock()
	entry, ok := c.cache[id]
	if !ok {
		// construct the new cache entry object
		entry = &CacheEntry{
			Method:  strings.ToUpper(request.Method),
			Url:     request.Url,
			Headers: make(http.Header),
			Body:    request.Body,
			Ttl:     time.Duration(request.Ttl) * time.Second,
			Hash:    fmt.Sprintf("%x", id),
		}
		for k, v := range request.Headers {
			entry.Headers.Add(k, v)
		}

		log.Printf("created new cache entry %s [url: %s, ttl: %s]",
			entry.Hash, entry.Url, entry.Ttl.String())

		// make the entry public, but lock it for accesss
		// start the fetch task
		entry.Fetching.Lock()
		c.cache[id] = entry
		go entry.task()
	}
	c.mutex.Unlock()

	// wait for the cache entry to be fully populated
	entry.Fetching.RLock()
	defer entry.Fetching.RUnlock()

	// return the response and error to the caller
	*r = entry.Response
	return entry.Error
}

// ---------------------------------------------------------------------------------------
//  private members
// ---------------------------------------------------------------------------------------

func (e *CacheEntry) task() {
	first := true
	client := http.Client{}

	for {
		// update
		response, err := e.update(&client)
		if err != nil {
			log.Println("failed to fetch url", err.Error())
		}

		// on the first run of the task the mutex is
		// already locked -> we dont need to lock it
		// ourself.
		if !first {
			e.Fetching.Lock()
		}
		e.Error = err
		e.Response = response
		e.Fetching.Unlock()

		first = false
		time.Sleep(e.Ttl)
	}
}

func (e *CacheEntry) update(client *http.Client) (string, error) {
	start := time.Now()

	// construct the new HTTP requests
	req, err := http.NewRequest(e.Method, e.Url, strings.NewReader(e.Body))
	if err != nil {
		return "", err
	}
	req.Header = e.Headers

	// perform the http request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// read the http servers response into a string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read body")
		return "", err
	}

	log.Println("fetched cache entry", e.Hash, "("+e.Url+")", "in", time.Since(start))

	return string(body), nil
}
