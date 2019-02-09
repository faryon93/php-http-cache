package metric

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
	"github.com/prometheus/client_golang/prometheus"
)

// ---------------------------------------------------------------------------------------
//  constants
// ---------------------------------------------------------------------------------------

const (
	Namespace = "http_cache"
)

// ---------------------------------------------------------------------------------------
//  imports
// ---------------------------------------------------------------------------------------

var (
	CacheHit = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "hit",
		Help:      "Total number of cache hits.",
	})

	CacheMiss = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "miss",
		Help:      "Total number of cache misses.",
	})

	CacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "size",
		Help:      "Number of entries in the cache table.",
	})
)

// ---------------------------------------------------------------------------------------
//  initializer
// ---------------------------------------------------------------------------------------

func init() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(CacheHit)
	prometheus.MustRegister(CacheMiss)
	prometheus.MustRegister(CacheSize)
}
