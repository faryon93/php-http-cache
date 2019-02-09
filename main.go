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
	"flag"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spiral/goridge"
)

// ---------------------------------------------------------------------------------------
//  variables
// ---------------------------------------------------------------------------------------

var (
	ForceColors bool
	Listen      string
)

// ---------------------------------------------------------------------------------------
//  application entry
// ---------------------------------------------------------------------------------------

func main() {
	cacheService := NewCacheService()

	// command line arguments
	flag.DurationVar(&cacheService.Timeout, "timeout", 2*time.Hour, "timeout for cache entrys")
	flag.StringVar(&Listen, "listen", ":6001", "rpc listen address")
	flag.BoolVar(&ForceColors, "colors", false, "force logging with colors")
	flag.Parse()

	// setup logger
	formater := logrus.TextFormatter{ForceColors: ForceColors, DisableColors: !ForceColors}
	logrus.SetFormatter(&formater)
	logrus.SetOutput(os.Stdout)

	logrus.Infoln("starting", GetAppVersion())

	ln, err := net.Listen("tcp", Listen)
	if err != nil {
		panic(err)
	}

	err = rpc.RegisterName("Cache", cacheService)
	if err != nil {
		panic(err)
	}

	go func() {
		// Expose the registered metrics via HTTP.
		http.Handle("/metrics", promhttp.Handler())
		logrus.Errorln(http.ListenAndServe(":6002", nil))
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go rpc.ServeCodec(goridge.NewCodec(conn))
	}
}
