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
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/faryon93/php-http-cache/metric"
)

// ---------------------------------------------------------------------------------------
//  constants
// ---------------------------------------------------------------------------------------

var (
	ErrInvalidTtl = errors.New("ttl should be at least 1s")
)

// ---------------------------------------------------------------------------------------
//  types
// ---------------------------------------------------------------------------------------

type CacheService struct {
	cache map[uint64]*CacheEntry
	mutex sync.RWMutex

	Timeout time.Duration
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
		logrus.Warnln("failed to decode command:", err.Error())
		return err
	}
	id := request.Hash()

	// check if there is already a cache entry present -> if not create a new one
	c.mutex.Lock()
	entry, ok := c.cache[id]
	if !ok {
		// construct the new cache entry object
		entry = &CacheEntry{
			Method:     strings.ToUpper(request.Method),
			Url:        request.Url,
			Body:       request.Body,
			Ttl:        time.Duration(request.Ttl) * time.Second,
			Id:         id,
			LastAccess: time.Now(),
			Response:   "",
		}

		// construct the header map
		entry.Headers, err = request.GetHeaders()
		if err != nil {
			c.mutex.Unlock()
			logrus.Errorln("rejecting request:", err.Error())
			return err
		}

		// make sure the ttl is not too low
		if entry.Ttl < time.Second {
			c.mutex.Unlock()
			logrus.Warnf("rejeting request: ttl (%s) to low", entry.Ttl.String())
			return ErrInvalidTtl
		}

		logrus.Infof("created new cache entry %s [url: %s, ttl: %s]",
			entry.String(), entry.Url, entry.Ttl.String())

		// make the entry public, but lock it for accesss
		// start the fetch task
		entry.Fetching.Lock()
		c.cache[id] = entry
		go entry.task(c)

		metric.CacheSize.Inc()
		metric.CacheMiss.Inc()

	} else {
		metric.CacheHit.Inc()
	}
	c.mutex.Unlock()

	// wait for the cache entry to be fully populated
	entry.Fetching.RLock()
	defer entry.Fetching.RUnlock()

	// return the response and error to the caller
	entry.LastAccess = time.Now()
	*r = entry.Response
	return entry.Error
}

func (c *CacheService) Remove(id uint64) {
	c.mutex.Lock()
	delete(c.cache, id)
	metric.CacheSize.Dec()
	c.mutex.Unlock()
}

// ---------------------------------------------------------------------------------------
//  private members
// ---------------------------------------------------------------------------------------

func (e *CacheEntry) task(service *CacheService) {
	first := true
	client := http.Client{}

	for {
		// stop the background fetching task after the configured timeout
		if service.Timeout > 0 && time.Now().After(e.LastAccess.Add(service.Timeout)) {
			service.Remove(e.Id)
			logrus.Infof("entry %s timed out: purging from cache", e.String())
			return
		}

		// fetch a fresh copy of the request body
		response, err := e.fetch(&client)
		if err != nil {
			logrus.Errorf("failed to fetch cache entry %s: %s",
				e.String(), err.Error())
		}

		// on the first run of the task the mutex is
		// already locked -> we dont need to lock it
		// ourself.
		if !first {
			e.Fetching.Lock()
		}

		// no error while fetching the response of the
		// requested resource
		if err == nil {
			e.Error = nil
			e.Response = response

		} else {
			// forward the fetching error only if there
			// isn't a cached version of the request
			if e.Response != "" {
				e.Error = err
				e.Response = ""
			}
		}

		e.Fetching.Unlock()

		first = false
		time.Sleep(e.Ttl)
	}
}

func (e *CacheEntry) fetch(client *http.Client) (string, error) {
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
		return "", err
	}

	return string(body), nil
}
