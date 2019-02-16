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
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/faryon93/php-http-cache/metric"
	"github.com/faryon93/php-http-cache/rpc"
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

// Cache service to be exported bei goridge.
type Service struct {
	cache map[uint64]*Entry
	mutex sync.RWMutex

	Timeout time.Duration
}

// ---------------------------------------------------------------------------------------
//  public functions
// ---------------------------------------------------------------------------------------

// NewService constructs a new cache service.
func NewService() *Service {
	return &Service{
		cache: make(map[uint64]*Entry),
	}
}

// ---------------------------------------------------------------------------------------
//  public members
// ---------------------------------------------------------------------------------------

// Request is used by PHP to issue a new cached request.
func (c *Service) Request(body string, r *string) error {
	// parse the request
	var request rpc.CmdRequest
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
		entry = &Entry{
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

// ---------------------------------------------------------------------------------------
//  private members
// ---------------------------------------------------------------------------------------

// Remove deletes a cache entry by its ID from the cache.
func (c *Service) remove(id uint64) {
	c.mutex.Lock()
	delete(c.cache, id)
	metric.CacheSize.Dec()
	c.mutex.Unlock()
}
