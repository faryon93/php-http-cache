package rpc

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
	"net/http"
	"strings"

	"github.com/cespare/xxhash"
)

// ---------------------------------------------------------------------------------------
//  imports
// ---------------------------------------------------------------------------------------

type CmdRequest struct {
	Method  string   `json:"method"`
	Url     string   `json:"url"`
	Body    string   `json:"body"`
	Headers []string `json:"headers"`
	Ttl     int      `json:"ttl"`
}

// ---------------------------------------------------------------------------------------
//  public members
// ---------------------------------------------------------------------------------------

func (r *CmdRequest) Hash() uint64 {
	headers := ""
	for _, header := range r.Headers {
		headers += header
	}

	return xxhash.Sum64String(r.Method + r.Url + r.Body + headers)
}

func (r *CmdRequest) GetHeaders() (http.Header, error) {
	headers := make(http.Header)

	for _, header := range r.Headers {
		h := strings.Split(header, ":")
		if len(h) != 2 {
			return nil, fmt.Errorf("header \"%s\" is badly formated", header)
		}

		headers.Add(strings.TrimSpace(h[0]), strings.TrimSpace(h[1]))
	}

	return headers, nil
}
