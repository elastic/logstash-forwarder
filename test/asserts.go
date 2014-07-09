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

package test

import (
	"reflect"
	"testing"
)

// REVU: TODO: move to kriterium

// ----------------------------------------------------------------------------
// Test helper functions
// ----------------------------------------------------------------------------

// Assert the equivalence of the expected and have arguments.
// Note that testing.T.Fatal is called on assert failure.
func AssertStringsEqual(t *testing.T, testname, resname string, expected, have string) {
	if expected != have {
		t.Fatalf("%s:%s - expected %q have %q", testname, resname, expected, have)
	}
}

func AssertEquals(t *testing.T, testname, resname string, expected, have interface{}) {
	vexp := reflect.ValueOf(expected)
	vhave := reflect.ValueOf(have)
	kexp := vexp.Kind()
	khave := vhave.Kind()
	if kexp != khave {
		t.Fatalf("'expected' and 'have' are not the same Kind", kexp, khave)
	}

}

func AssertNotNil(t *testing.T, testname, resname string, ref interface{}) {
	ok := true
	switch t := ref.(type) {
	case string:
		ok = t != ""
	case error:
		ok = t != nil
	default:
		ok = t != nil
	}
	if !ok {
		t.Fatalf("%s:%s is nil", testname, resname)
	}
}

func AssertNil(t *testing.T, testname, resname string, ref interface{}) {
	ok := true
	switch t := ref.(type) {
	case string:
		ok = t == ""
	case error:
		ok = t == nil
	default:
		ok = t == nil
	}
	if !ok {
		t.Fatalf("%s:%s is not nil: %q", testname, resname, ref)
	}
}

// ----------------------------------------------------------------------------
// Unit Test Assertions
// ----------------------------------------------------------------------------

type Assertion interface {
	StringsEqual(label string, expected, have string)
	NotNil(label string, v interface{})
	Nil(label string, v interface{})
	SameReference(label string, expected, have interface{})
}

func GetAssertionFor(t *testing.T, testName string) Assertion {
	if t == nil {
		panic("BUG: t is nil")
	}
	if testName == "" {
		panic("BUG: testName is nil")
	}
	return &assertion{t, testName}
}

type assertion struct {
	t        *testing.T
	testName string
}

func (t *assertion) StringsEqual(label string, expected, have string) {
	AssertStringsEqual(t.t, t.testName, label, expected, have)
}

func (t *assertion) SameReference(label string, expected, have interface{}) {
	AssertEquals(t.t, t.testName, label, expected, have)
}

func (t *assertion) NotNil(label string, v interface{}) {
	AssertNotNil(t.t, t.testName, label, v)
}

func (t *assertion) Nil(label string, v interface{}) {
	AssertNil(t.t, t.testName, label, v)
}
