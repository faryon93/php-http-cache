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

func (e *Entry) IsExpired(timeout time.Duration) bool {
	return time.Now().After(e.LastAccess.Add(timeout))
}

func (e *Entry) String() string {
	return fmt.Sprintf("%x", e.Id)
}

// ---------------------------------------------------------------------------------------
//  private members
// ---------------------------------------------------------------------------------------

// Task periodically updates the requests response in the cache.
func (e *Entry) task(service *Service) {
	first := true
	client := http.Client{}

	for {
		// stop the background fetching task after the configured timeout
		if service.Timeout > 0 && e.IsExpired(service.Timeout) {
			service.remove(e.Id)
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

func (e *Entry) fetch(client *http.Client) (string, error) {
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
