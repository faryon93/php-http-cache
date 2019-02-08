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
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fastjson"
)

// ---------------------------------------------------------------------------------------
//  types
// ---------------------------------------------------------------------------------------

type CacheService struct {
	Cache map[string]*CacheEntry
	mutex sync.RWMutex
}

type CacheEntry struct {
	Method string
	Url    string
	Body   string
	Ttl    time.Duration
	Error  error

	Fetching sync.RWMutex
	Response string
}

// ---------------------------------------------------------------------------------------
//  public functions
// ---------------------------------------------------------------------------------------

func NewCacheService() *CacheService {
	return &CacheService{
		Cache: make(map[string]*CacheEntry),
	}
}

// ---------------------------------------------------------------------------------------
//  public members
// ---------------------------------------------------------------------------------------

func (c *CacheService) Request(body string, r *string) error {
	// parse the command
	cmd, err := fastjson.Parse(body)
	if err != nil {
		return err
	}

	method := string(cmd.GetStringBytes("method"))
	url := string(cmd.GetStringBytes("url"))
	ttl := time.Duration(cmd.GetInt("ttl")) * time.Second
	reqBody := string(cmd.GetStringBytes("body"))
	log.Println(cmd)

	c.mutex.Lock()
	entry, ok := c.Cache[url]
	if !ok {
		entry = &CacheEntry{
			Method: strings.ToUpper(method),
			Body:   reqBody,
			Url:    url,
			Ttl:    ttl,
		}

		entry.Fetching.Lock()
		c.Cache[url] = entry
		log.Printf("cache miss: creating new entry [url: %s, ttl: %s]", url, ttl.String())
		go entry.task()
	}
	c.mutex.Unlock()

	entry.Fetching.RLock()
	*r = entry.Response
	log.Println("returning entry entry", url)
	entry.Fetching.RUnlock()

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

	log.Println("updating cache entry", e.Url)

	// construct the new HTTP requests
	req, err := http.NewRequest(e.Method, e.Url, strings.NewReader(e.Body))
	if err != nil {
		return "", err
	}

	// TODO: use headers sent by the calling application
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "XOrbit.de Connect")

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

	log.Println("fetched", e.Url, "in", time.Since(start))

	return string(body), nil
}
