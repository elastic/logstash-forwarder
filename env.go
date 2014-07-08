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
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"log"
	"lsf/schema"
	"lsf/system"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// Base directory of an LSF base
const RootDir = ".lsf"

// ----------------------------------------------------------------------------
// Environment Types
// ----------------------------------------------------------------------------

type varKey string

func VarKey(v string) varKey {
	return varKey(v)
}

const (
	VarHomePath     varKey = "lsf.home.path"
	VarHomePort            = "lsf.home.port"
	VarConfig              = "lsf.config"
	VarHomeFileInfo        = "lsf.home.fileinfo"
	//	VarUserSigChan         = "user.signal.channel"
	VarSupervisor = "lsf.process.supervisor"
)

// ----------------------------------------------------------------------------
// LSF Runtime Environment
// ----------------------------------------------------------------------------

// LSF runtime environment for commands and processes, provides a managed
// interface to system resources for concurrent goroutines.
type Environment struct {
	lock     sync.RWMutex
	bound    bool
	varslock sync.RWMutex
	vars     map[varKey]interface{} // runtime vars only

	registrar system.Registrar
	docs      map[string]system.Document
	docslock  sync.RWMutex
	remotes   map[string]*schema.Port
	streams   map[string]*schema.LogStream
	journals  map[string]*schema.LogJournal
}

// Initializes the structural members of Environment.
func NewEnvironment() *Environment {
	env := &Environment{
		bound:    false,
		vars:     make(map[varKey]interface{}),
		docs:     make(map[string]system.Document),
		streams:  make(map[string]*schema.LogStream),
		remotes:  make(map[string]*schema.Port),
		journals: make(map[string]*schema.LogJournal),
	}
	return env
}

// --------------------------------------------------------------
// Environment: Life Cycle
// --------------------------------------------------------------

// Creates a new LSF environment in directory path.
// Path must be an absolute path.
// Sets state flag for env.
// Returns error if called on existing environemnt at path.
func CreateEnvironment(dir string, force bool) (rootpath string, err error) {
	if !IsAbsPath(dir) {
		return "", ERR.RELATIVE_PATH("CreateEnvironment:")
	}

	defer panics.Recover(&err)

	// is the working dir the same as user home?
	userHome := system.UserHome()
	isUserHome := userHome == dir

	// onetime setup user account global LS/F port
	// if not existing
	if !isUserHome {
		// create user level .lsf environment if not existing
		//		if _, e := CreateEnvironment(userHome, false); e != nil && e != E_EXISTING_LSF {
		if _, e := CreateEnvironment(userHome, false); e != nil && !ERR.EXISTING_LSF.Matches(e) {
			return "", e
		}
	}

	// determine LSF environment root path.
	// overwrite of existing LSF environment must be forced.
	root := rootAt(dir)
	exists := exists(root)
	if exists && !force {
		return "", ERR.EXISTING_LSF()
	}

	// lock out all others for this op
	uid := HexShaDigest(dir) // unique id for the environment based on its absolute path
	var portPath string
	switch isUserHome {
	case true:
		portPath = path.Join(userHome)
	default:
		portPath = path.Join(userHome, RootDir)
	}

	opLock, _, e := system.ExclusiveResourceOp(portPath, system.Op.LsfNew, uid, "new environment")
	panics.OnError(e, "CreateEnvironment")
	defer opLock.Unlock()

	// clean start
	// nop for new - meaningful only if existing
	e = os.RemoveAll(root)
	panics.OnError(e, "Environment.CreateEnvironment", "os.RemoveAll", "root:", root)

	// create the minimal structure
	e = os.Mkdir(root, os.ModeDir|defaultDirMode)
	panics.OnError(e, "Environment.CreateEnvironment", "os.Mkdir", "root:", root)

	// and system meta data
	docId := string("system")
	data := map[string][]byte{
		"create-time": []byte(time.Now().String()),
	}

	registrar, e := system.StartRegistry(root)
	panics.OnError(e, "Environment.CreateEnvironment", "system.StartRegistry", "root:", root)
	//	defer func() { registrar.Signal() <- struct{}{} }() // stop the registrar on return

	_, e = registrar.CreateDocument(docId, data)
	panics.OnError(e, "Environment.CreateEnvironment", "registrar.CreateDocument", "docId:", docId)

	// stop the registrar
	registrar.Signal() <- struct{}{}

	return root, nil
}

func (env *Environment) Shutdown() error {

	if env == nil {
		panic("BUG - Environment.Shutdown: env is nil")
	}

	env.lock.Lock()
	defer env.lock.Unlock()

	if !env.bound {
		return ERR.ILLEGAL_STATE()
	}

	if registrar := env.registrar; registrar != nil {
		registrar.Signal() <- struct{}{}
		<-registrar.Status()
	}

	//	port, _ := env.Get(VarHomePort)
	//	log.Printf("DEBUG: Environment.Shutdown: %s", port.(*Port).Path())

	return nil
}

func (env *Environment) Initialize(dir string) (err error) {

	defer panics.Recover(&err)

	env.lock.Lock()
	defer env.lock.Unlock() // TODO: these need deadlines.

	if env.bound { // REVU: shouldn't this be an error?
		return nil
	}
	if dir == "" {
		return ERR.ILLEGAL_ARGUMENT("dir", "zerovalue")
	}

	if !IsAbsPath(dir) {
		dir = path.Join(Wd(), dir)
	}
	root := rootAt(dir)

	// check if exists
	if !exists(root) {
		return ERR.NOT_EXISTING_LSF()
	}

	env.bound = true
	port, e := schema.NewLocalPort(root)
	panics.OnError(e, "Environment.Initialize:", "schema.NewLocalPort", "root:", root)

	env.Set(VarHomePort, port) // panics

	e = env.startRegistrar()
	panics.OnError(e, "Environment.Initialize:", "env.startRegistrar")

	sysdoc := string("system")
	env.loadDocuments([]string{sysdoc})

	_, e = env.registrar.ReadDocument(sysdoc)
	panics.OnError(e, "Environment.Initialize:", "env.ReadDocument")

	return nil
}

func (env *Environment) startRegistrar() (err error) {
	defer panics.Recover(&err)

	// REVU: what's the issue? why not just ignore it?
	if env.registrar != nil {
		return ERR.ILLEGAL_STATE_REGISTRAR_RUNNING()
	}

	port, found := env.Get(VarHomePort)
	panics.OnFalse(found, "BUG", VarHomePort, "not set")

	home := port.(*schema.Port).Address()
	registrar, e := system.StartRegistry(home)
	panics.OnError(e)

	env.registrar = registrar

	return nil
}

// --------------------------------------------------------------
// Environment: Properties & what not
// --------------------------------------------------------------

// Returns the path of the LSF Port to which the environment
// is bound.
func (env *Environment) Port() string {
	if !env.bound {
		panic("BUG - env not bound")
	}
	v, found := env.Get(VarHomePort)
	if !found {
		return ""
	}
	return v.(*schema.Port).Address()
}

func (env *Environment) IsBound() bool {
	env.lock.Lock()
	defer env.lock.Unlock()

	return env.bound
}

// TEMP TODO REMOVE
func (env *Environment) Debug() {
	log.Printf("debug- in env.go\n")
	log.Printf("bound:   %t\n", env.bound)
	//	log.Printf("created: %s\n", env.created)
	//	log.Printf("updated: %s\n", env.updated)
	log.Printf("vars: %s\n", env.vars)
}

// Returns the absolute id of a resource in context
// of this environment. The argument 'name' is a
// relative identifier.
// This routine will panic if called from an unbound Environment.
func (env *Environment) ResourceId(name string) string {
	if !env.bound {
		panic("BUG - env not bound")
	}
	v, found := env.Get(VarHomePort)
	panics.OnFalse(found, "BUG", "Environment.ResourceId", "VarHomePort not bound!")

	return path.Join(v.(*schema.Port).Address(), name)
}

// Returns true if an LSF environemnt exists in the given path
func (env *Environment) Exists(path string) bool {
	return exists(rootAt(path))
}

func exists(path string) bool {
	_, e := os.Stat(path)
	if e != nil {
		return false
	}
	return true
}

func rootAt(dir string) string {
	return path.Join(dir, RootDir)
}

// panics
func (env *Environment) GetResourceIds(restype string) []string {
	root := env.Port()
	dir, e := os.Open(path.Join(root, restype))
	if e != nil {
		return []string{}
	}

	dirnames, e := dir.Readdirnames(0)
	// if resource type dir exists and is empty then we have a bug
	panics.OnError(e, "Environment.GetResourceIds:", restype, "BUG - directory is empty", dir.Name())

	resIds := make([]string, len(dirnames))
	i := 0
	for _, dirname := range dirnames {
		if dirname[0] != '.' {
			resIds[i] = dirname
			i++
		}
	}
	return resIds[:i]
}

func (env *Environment) ExclusiveResourceOp(opcode system.OpCode, resId string, meta string) (opLock system.Lock, lockId string, err error) {
	return system.ExclusiveResourceOp(env.Port(), opcode, resId, meta)
}

func (env *Environment) GetResourceDigests(restype string, verbose bool, encoder system.DocumentDigestFn) []string {

	resourceIds := env.GetResourceIds(restype)

	var digest digestFn = justResourceId
	if verbose {
		digest = digestForResourceId
	}

	digests := make([]string, len(resourceIds))
	for i, resid := range resourceIds {
		digests[i] = digest(env, restype, resid, encoder)
	}
	return digests
}

type digestFn func(env *Environment, restype string, resid string, encode system.DocumentDigestFn) string

// See GetResourceDigests()
func justResourceId(env *Environment, restype string, resid string, encode system.DocumentDigestFn) string {
	return resid
}

// See GetResourceDigests()
func digestForResourceId(env *Environment, restype string, resid string, encode system.DocumentDigestFn) string {
	docId := string(fmt.Sprintf("%s.%s.%s", restype, resid, restype))
	doc, e := env.LoadDocument(docId)
	panics.OnError(e, "BUG", "getResourceDigests:", "loadDocument", docId)
	panics.OnTrue(doc == nil, "BUG", "getResourceDigests:", "loadDocument", docId)
	return encode(doc)
}

// --------------------------------------------------------------
// Environment: Variables
// --------------------------------------------------------------

// REVU: the found retval is buggin me..
// TODO: remove it and guard in Set against nils. nil v => not found
func (env *Environment) Get(key varKey) (v interface{}, found bool) {
	env.varslock.RLock()
	defer env.varslock.RUnlock()

	v, found = env.vars[key]
	return
}

// nil value not accepted.
func (env *Environment) Set(key varKey, v interface{}) (prev interface{}, e error) {
	if v == nil {
		return nil, ERR.ILLEGAL_ARGUMENT("Environment.Set", "v", "nil value")
	}

	env.varslock.Lock()
	defer env.varslock.Unlock()

	prev, _ = env.vars[key]
	env.vars[key] = v
	return prev, nil
}

// --------------------------------------------------------------
// Environment: System Documents
// --------------------------------------------------------------

// REVU: foolishness
var defaultDirMode = os.FileMode(0755)

// Creates a document in the bound LSF Port.
func (env *Environment) CreateDocument(docId string, datamap system.DataMap) error {
	if !env.bound {
		return ERR.ILLEGAL_STATE()
	}
	mappings := datamap.Mappings()
	_, e := env.registrar.CreateDocument(docId, mappings)
	if e != nil {
		return e
	}
	return nil
}

// Update fully flushes the document back to the bound LSF Port.
// REVU TODO clarify ok/error - do we need both?
func (env *Environment) UpdateDocument(doc system.Document) (ok bool, err error) {
	defer panics.Recover(&err)

	ok, e := env.registrar.UpdateDocument(doc)
	panics.OnError(e, "Environment.UpdateDocument", "docId:", doc.Id())
	panics.OnFalse(ok, "Environment.UpdateDocument", "docId:", doc.Id())

	env.docslock.Lock()
	env.docs[doc.Id()] = doc
	env.docslock.Unlock()

	return ok, e
}

// Load fully reads the identified document from the bound LSF Port.
func (env *Environment) LoadDocument(docId string) (doc system.Document, err error) {
	defer panics.Recover(&err)

	doc, e := env.registrar.ReadDocument(docId)
	panics.OnError(e, "Environment.LoadDocument", "docId:", docId)

	env.docslock.Lock()
	env.docs[docId] = doc
	env.docslock.Unlock()

	return doc, e
}

// Get document by id. Loads the document if not already loaded.
func (env *Environment) GetDocument(docId string) (doc system.Document, err error) {
	env.docslock.Lock()
	doc = env.docs[docId]
	env.docslock.Unlock()

	if doc == nil {
		return env.LoadDocument(docId)
	}

	return doc, nil
}

func (env *Environment) AddSystemDocument(opcode system.OpCode, id, docId, meta string, datamap system.DataMap) error {
	opLock, _, e := system.ExclusiveResourceOp(env.Port(), opcode, id, meta)
	if e != nil {
		return e
	}
	defer opLock.Unlock()

	// check if exists
	doc, e := env.LoadDocument(docId)
	if e == nil && doc != nil {
		return ERR.EXISTING("AddSystemDocument:")
	}

	// create it
	e = env.CreateDocument(docId, datamap)
	if e != nil {
		return e
	}

	return nil
}

func (env *Environment) UpdateSystemDocument(opcode system.OpCode, id, docId, meta string, updates map[string][]byte) error {
	// do not permit concurrent updates to this stream
	opLock, _, e := system.ExclusiveResourceOp(env.Port(), opcode, id, meta)
	if e != nil {
		return e
	}
	defer opLock.Unlock()

	// verify it exists
	doc, e := env.LoadDocument(docId)
	if e != nil || doc == nil {
		return ERR.NOT_EXISTING()
	}

	previous := doc.SetAll(updates)
	if len(previous) == 0 {
		return fmt.Errorf("warning: no changes were made to document %s", docId)
	}

	ok, e := env.UpdateDocument(doc)
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("failed to update document: %s", docId)
	}

	return nil
}

func (env *Environment) RemoveSystemDocument(opcode system.OpCode, id, docId, meta string) error {
	// NOTE: ops that require stream to exist can also lock this op
	opLock, _, e := system.ExclusiveResourceOp(env.Port(), opcode, id, meta)
	if e != nil {
		return e
	}
	defer opLock.Unlock()

	// check existing
	doc, e := env.LoadDocument(docId)
	if e != nil || doc == nil {
		return ERR.NOT_EXISTING()
	}

	// remove doc
	ok, e := env.DeleteDocument(docId)
	if e != nil {
		return e
	}
	if !ok {
		return fmt.Errorf("failed to delete document: %s", docId)
	}

	// remove the stream directory from the lsf environment
	docpath, _ := system.DocpathForKey(env.Port(), docId)
	e = os.RemoveAll(docpath)
	if e != nil {
		return e
	}

	return nil
}

// Deletes the document from the environemnt and the bound LSF Port.
// REVU TODO clarify ok/error - do we need both?
func (env *Environment) DeleteDocument(docId string) (ok bool, err error) {
	defer panics.Recover(&err)

	ok, e := env.registrar.DeleteDocument(docId)
	panics.OnError(e, "Environment.DeleteDocument", "docId:", docId)
	panics.OnFalse(ok, "Environment.DeleteDocument", "docId:", docId)

	env.docslock.Lock()
	delete(env.docs, docId)
	env.docslock.Unlock()

	return ok, e
}

// All documents (ids) are presumed to be valid in context of the bound LSF Port.
// Returns error (and stops loading) on missing doc(s).
func (env *Environment) loadDocuments(docIds []string) (err error) {

	defer panics.Recover(&err)

	env.docslock.Lock()
	defer env.docslock.Unlock()

	for _, docId := range docIds {
		_, found := env.docs[docId]
		if !found {
			doc, e := env.registrar.ReadDocument(docId)
			panics.OnError(e, "Environment.loadDocuments", "docId:", docId)
			env.docs[docId] = doc
		}
	}
	return nil
}

func getRecordHierarchy(record string) (documents []string, key string, err error) {
	terms := strings.Split(record, ".")
	n := len(terms)
	if n < 2 {
		return nil, "", ERR.ILLEGAL_ARGUMENT("record")
	}

	docname := terms[n-2]
	key = terms[n-1]

	docs := make([]string, n-1)
	docs[0] = string(docname)
	for i := 1; i < n-1; i++ {
		docs[i] = string(strings.Join(terms[0:i], ".") + "." + docname)
	}
	return docs, key, nil
}

// record is interpreted as a dot notation path. final term
// is record key in the document in the path. The simplest
// record spec is "docname.recname". A record arg that does
// not have at least 2 parts is rejected as ERR.INVALID().
//
// GetRecord side-effects:
// A call to this method will load the entire doc that contains
// the record. Additionally, the port directory is walked up
// from the found document and each matching doc is loaded.
//
// The final value for key in record reflects the hierarchical
// scoping of the matching documents.
//
// if not found, will return nil, nil.
func (env *Environment) GetRecord(record string) (value []byte, err error) {

	defer panics.Recover(&err)

	if !env.bound {
		return nil, ERR.ILLEGAL_STATE()
	}

	documents, key, e := getRecordHierarchy(record)
	panics.OnError(e, "Environment.GetRecord:", "record:", record)

	e = env.loadDocuments(documents)
	panics.OnError(e, "Environment.GetRecord:")

	value = env.resolveRecord(documents, key)
	return value, nil
}

// A document Record is resolved in context of the resource hierarchy.
// The record is logically identified by the key parameter.
// The set of documents sorted from global to local scope.
// Resolution of the record is to match the value from the most proximate
// (i.e. local) document provided.
func (env *Environment) resolveRecord(documents []string, key string) []byte {
	env.docslock.RLock()
	defer env.docslock.RUnlock()

	var value []byte
	for _, docId := range documents {
		doc, found := env.docs[docId]
		if found {
			if v := doc.Get(key); v != nil {
				value = v
			}
		}
	}
	return value
}

// --------------------------------------------------------------
// Log Streams
// --------------------------------------------------------------

func (env *Environment) UpdateLogStream(id string, updates map[string][]byte) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	docId := fmt.Sprintf("stream.%s.stream", id)
	return env.UpdateSystemDocument(system.Op.StreamUpdate, id, docId, "stream-update", updates)
}

func (env *Environment) RemoveLogStream(id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	docId := fmt.Sprintf("stream.%s.stream", id)
	return env.RemoveSystemDocument(system.Op.StreamRemove, id, docId, "stream-remove")
}

func (env *Environment) AddLogStream(id, basepath, pattern, journalModel string, fields map[string]string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}
	if basepath == "" {
		return fmt.Errorf("basepath is empty")
	}
	if pattern == "" {
		return fmt.Errorf("pattern is empty")
	}
	if journalModel == "" {
		return fmt.Errorf("journalModel is empty")
	}

	docId := fmt.Sprintf("stream.%s.stream", id)

	// TODO: this needs to be checked - valid journal mode value?
	mode := schema.ToJournalModel(journalModel)
	logStream := schema.NewLogStream(id, basepath, mode, pattern, fields)

	return env.AddSystemDocument(system.Op.StreamAdd, id, docId, "stream-add", logStream)
}

// --------------------------------------------------------------
// Remote Ports
// --------------------------------------------------------------

func (env *Environment) RemoveRemotePort(id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	docId := fmt.Sprintf("remote.%s.remote", id)
	return env.RemoveSystemDocument(system.Op.RemoteRemove, id, docId, "remote-remove")
}

func (env *Environment) UpdateRemotePort(id string, updates map[string][]byte) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	docId := fmt.Sprintf("remote.%s.remote", id)
	return env.UpdateSystemDocument(system.Op.RemoteUpdate, id, docId, "remote-update", updates)
}

func (env *Environment) AddRemotePort(id, host string, port int) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}
	if host == "" {
		return fmt.Errorf("host is empty")
	}
	if port <= 0 {
		return fmt.Errorf("invalid port number") // TODO: test for the actual non-reserved range
	}

	docId := fmt.Sprintf("remote.%s.remote", id)
	remotePort, e := schema.NewRemotePort(id, host, port)
	if e != nil {
		return e
	}

	return env.AddSystemDocument(system.Op.RemoteAdd, id, docId, "remote-add", remotePort)
}
