package cache

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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------------------------
//  types
// ---------------------------------------------------------------------------------------

type Entry struct {
	Method  string
	Url     string
	Body    string
	Headers http.Header
	Ttl     time.Duration

	LastAccess time.Time

	Fetching sync.RWMutex
	Response string
	Error    error
	Id       uint64
}

// ---------------------------------------------------------------------------------------
//  public members
// ---------------------------------------------------------------------------------------

func (c *Entry) String() string {
	return fmt.Sprintf("%x", c.Id)
}

// ---------------------------------------------------------------------------------------
//  private members
// ---------------------------------------------------------------------------------------

// Task periodically updates the requests response in the cache.
func (c *Entry) task(service *Service) {
	first := true
	client := http.Client{}

	for {
		// stop the background fetching task after the configured timeout
		if service.Timeout > 0 && time.Now().After(c.LastAccess.Add(service.Timeout)) {
			service.remove(c.Id)
			logrus.Infof("entry %s timed out: purging from cache", c.String())
			return
		}

		// fetch a fresh copy of the request body
		response, err := c.fetch(&client)
		if err != nil {
			logrus.Errorf("failed to fetch cache entry %s: %s",
				c.String(), err.Error())
		}

		// on the first run of the task the mutex is
		// already locked -> we dont need to lock it
		// ourself.
		if !first {
			c.Fetching.Lock()
		}

		// no error while fetching the response of the
		// requested resource
		if err == nil {
			c.Error = nil
			c.Response = response

		} else {
			// forward the fetching error only if there
			// isn't a cached version of the request
			if c.Response != "" {
				c.Error = err
				c.Response = ""
			}
		}

		c.Fetching.Unlock()

		first = false
		time.Sleep(c.Ttl)
	}
}

func (c *Entry) fetch(client *http.Client) (string, error) {
	// construct the new HTTP requests
	req, err := http.NewRequest(c.Method, c.Url, strings.NewReader(c.Body))
	if err != nil {
		return "", err
	}
	req.Header = c.Headers

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
