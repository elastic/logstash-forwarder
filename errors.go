// Licensed to Elasticsearch under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package lsf

import (
	"github.com/elasticsearch/kriterium/errors"
)

// ----------------------------------------------------------------------------
// error codes
// ----------------------------------------------------------------------------

var ERR = struct {
	USAGE,
	ILLEGAL_STATE,
	ILLEGAL_ARGUMENT,
	OP_FAILURE,
	RELATIVE_PATH,
	EXISTING_LSF,
	NOT_EXISTING_LSF,
	EXISTING,
	NOT_EXISTING,
	ILLEGAL_STATE_REGISTRAR_RUNNING,
	EXISTING_STREAM,
	CONCURRENT errors.TypedError
}{
	USAGE:                           errors.USAGE,
	ILLEGAL_ARGUMENT:                errors.ILLEGAL_ARGUEMENT,
	ILLEGAL_STATE:                   errors.ILLEGAL_STATE,
	OP_FAILURE:                      errors.New("operation failed"),
	ILLEGAL_STATE_REGISTRAR_RUNNING: errors.New("registrar already running"),                   // REVU: illegal state..
	RELATIVE_PATH:                   errors.New("path is not absolute"),                        // REVU: isn't this just an illegal arg error?
	EXISTING_LSF:                    errors.New("lsf environment already exists"),              // REVU: illegal state..
	NOT_EXISTING_LSF:                errors.New("lsf environment does not exists at location"), // REVU: illegal state..
	EXISTING:                        errors.New("lsf resource already exists"),                 // REVU: illegal state..
	NOT_EXISTING:                    errors.New("lsf resource does not exist"),                 // REVU: illegal state..
	EXISTING_STREAM:                 errors.New("stream already exists"),                       // REVU: illegal state..
	CONCURRENT:                      errors.New("concurrent operation error"),                  // REVU: illegal state..
}

var WARN = struct {
	NO_OP errors.TypedError
}{
	NO_OP: errors.New("warning: nop"),
}
