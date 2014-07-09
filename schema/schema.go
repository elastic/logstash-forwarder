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

package schema

import (
	"time"
)

// defines the generic logical form of all schema entities
type defmapping map[string][]byte

// all schema entities support this interface.
type Encoder interface {
	Mappings() map[string][]byte
	String() string
	Bytes() []byte
}
type Decoder interface {
	Map(mapping defmapping) interface{}
}

func (data defmapping) Mappings() map[string][]byte {
	return data
}

var DefaultSystemMappings = defmapping{}

// ----------------------------------------------------------------------
// LogRecord
// ----------------------------------------------------------------------

type LogRecord struct {
	stream    string
	command   string
	Timestamp time.Time
}

// ----------------------------------------------------------------------
// JournalModel
// ----------------------------------------------------------------------

type journalModel string

// enum
var JournalModel = struct {
	Rotation, Rollover journalModel
}{
	Rotation: "rotation",
	Rollover: "rollover",
}

func ToJournalModel(v string) journalModel {
	switch journalModel(v) {
	case JournalModel.Rotation:
		return JournalModel.Rotation
	case JournalModel.Rollover:
		return JournalModel.Rollover
	}
	return journalModel("")
}

// ----------------------------------------------------------------------
// Identifiers
// ----------------------------------------------------------------------

// The Stash - universal journal schema for log streams
type LogJournal struct {
	Stream  string
	Entries []*LogJournalEntry
}

// An entry in the LogJournal
type LogJournalEntry struct {
	Timestamp time.Time
	Data      []byte // REVU: TODO: consider the string..
}
