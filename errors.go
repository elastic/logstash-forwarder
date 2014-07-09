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
	Usage,
	IllegalArgument,
	IllegalState,
	OpFailure,
	LsfEnvironmentExists,
	LsfEnvironmentDoesNotExist,
	ResourceExists,
	ResourceDoesNotExist,
	_stub            errors.TypedError
}{
	Usage:                      errors.Usage,
	IllegalArgument:            errors.IllegalArgument,
	IllegalState:               errors.IllegalState,
	OpFailure:                  errors.New("operation failed"),
	LsfEnvironmentExists:       errors.New("lsf environment already exists"),
	LsfEnvironmentDoesNotExist: errors.New("lsf environment does not exists at location"),
	ResourceExists:             errors.New("lsf resource already exists"),
	ResourceDoesNotExist:       errors.New("lsf resource does not exist"),
}

var WARN = struct {
	NoOp errors.TypedError
}{
	NoOp: errors.New("warning: no op"),
}
