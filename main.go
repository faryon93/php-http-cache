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
	"net/rpc"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spiral/goridge"
)

// ---------------------------------------------------------------------------------------
//  variables
// ---------------------------------------------------------------------------------------

var (
	ForceColors bool
)

// ---------------------------------------------------------------------------------------
//  application entry
// ---------------------------------------------------------------------------------------

func main() {
	cacheService := NewCacheService()

	// command line arguments
	flag.DurationVar(&cacheService.Timeout, "timeout", 2*time.Hour, "timeout for cache entrys")
	flag.BoolVar(&ForceColors, "colors", false, "force logging with colors")
	flag.Parse()

	// setup logger
	formater := logrus.TextFormatter{ForceColors: ForceColors, DisableColors: !ForceColors}
	logrus.SetFormatter(&formater)
	logrus.SetOutput(os.Stdout)

	ln, err := net.Listen("tcp", ":6001")
	if err != nil {
		panic(err)
	}

	err = rpc.RegisterName("Cache", cacheService)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeCodec(goridge.NewCodec(conn))
	}
}
