package lsf

import (
	"errors"
	"fmt"
	"log"
	"lsf/schema"
	"lsf/system"
	"lsf/anomaly"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// Base directory of an LSF base
const RootDir = ".lsf"

var E_USAGE = errors.New("invalid command usage")
var E_INVALID = errors.New("invalid argument")
var E_RELATIVE_PATH = errors.New("path is not absolute")
var E_EXISTING_LSF = errors.New("lsf environment already exists")
var E_NOTEXISTING_LSF = errors.New("lsf environment does not exists at location")
var E_EXISTING = errors.New("lsf resource already exists")
var E_NOTEXISTING = errors.New("lsf resource does not exist")
var E_ILLEGALSTATE = errors.New("illegal state")
var E_ILLEGALSTATE_REGISTRAR_RUNNING = errors.New("Registrar already running")
var E_EXISTING_STREAM = errors.New("stream already exists")
var E_CONCURRENT = errors.New("concurrent operation error")

// keep it package private
type varKey string

const (
	VarHomePath     varKey = "lsf.home.path"
	VarHomePort            = "lsf.home.port"
	VarConfig              = "lsf.config"
	VarHomeFileInfo        = "lsf.home.fileinfo"
)

// Restrict what can go in Var map
type EnvId string

// LSF Runtime environment
type Environment struct {
	lock     sync.RWMutex
	bound    bool
	varslock sync.RWMutex
	vars     map[varKey]interface{} // runtime vars only

	registrar system.Registrar
	docs      map[system.DocId]system.Document
	docslock  sync.RWMutex
	remotes   map[string]*schema.Port
	streams   map[schema.StreamId]*schema.LogStream
	journals  map[string]*schema.LogJournal
}

func NewEnvironment() *Environment {
	env := &Environment{
		bound:    false,
		vars:     make(map[varKey]interface{}),
		docs:     make(map[system.DocId]system.Document),
		streams:  make(map[schema.StreamId]*schema.LogStream),
		remotes:  make(map[string]*schema.Port),
		journals: make(map[string]*schema.LogJournal),
	}
	return env
}

func (env *Environment) Port() string {
	if !env.bound {
		panic("BUG - env not bound")
	}
	v, found := env.Get(VarHomePort)
	if !found {
		return ""
	}
	return v.(*schema.Port).Path()
}
func (env *Environment) ResourceId(name string) string {
	if !env.bound {
		panic("BUG - env not bound")
	}
	v, found := env.Get(VarHomePort)
	if !found {
		panic("BUG - home port should be bound")
	}
	return path.Join(v.(*schema.Port).Path(), name)
}

func (env *Environment) CreateDocument(docid system.DocId, datamap system.DataMap) error {
	if !env.bound {
		return E_ILLEGALSTATE
	}
	mappings := datamap.Mappings()
	//	mappings["update-time"]=[]byte(time.Now().String())
	_, e := env.registrar.CreateDocument(docid, mappings)
	if e != nil {
		return e
	}
	return nil
}

// record is interpreted as a dot notation path. final term
// is record key in the document in the path. The simplest
// record spec is "docname.recname". A record arg that does
// not have at least 2 parts is rejected as E_INVALID.
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
func (env *Environment) GetRecord(record string) ([]byte, error) {
	//	log.Println("----------------------------------------")
	if !env.bound {
		return nil, E_ILLEGALSTATE
	}

	documents, key, e := getRecordHierarchy(record)
	if e != nil {
		return nil, e
	}

	env.loadDocuments(documents)

	value := env.resolveRecord(documents, key)
	return value, nil
}

func (env *Environment) resolveRecord(documents []system.DocId, key string) []byte {
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

func (env *Environment) DeleteDocument(docid system.DocId) (bool, error) {
	ok, e := env.registrar.DeleteDocument(docid)
	if e == nil && ok {
		env.docslock.Lock()
		delete(env.docs, docid)
		env.docslock.Unlock()
	}
	return ok, e
}

func (env *Environment) UpdateDocument(doc system.Document) (bool, error) {
	ok, e := env.registrar.UpdateDocument(doc)
	if e == nil && ok {
		env.docslock.Lock()
		env.docs[doc.Id()] = doc
		env.docslock.Unlock()
	}
	return ok, e
}

func (env *Environment) LoadDocument(docid system.DocId) (system.Document, error) {
	doc, e := env.registrar.ReadDocument(docid)
	if e == nil {
		env.docslock.Lock()
		env.docs[docid] = doc
		env.docslock.Unlock()
	}
	return doc, e
}
func (env *Environment) loadDocuments(documents []system.DocId) {
	env.docslock.Lock()
	defer env.docslock.Unlock()

	for _, docid := range documents {
		_, found := env.docs[docid]
		if !found {
			doc, e := env.registrar.ReadDocument(docid)
			if e == nil {
				//				log.Printf("DEBUG: Environment.loadDocuments: loaded %q", docid)
				env.docs[docid] = doc
			}
		}
	}
}

func getRecordHierarchy(record string) (documents []system.DocId, key string, err error) {
	terms := strings.Split(record, ".")
	n := len(terms)
	if n < 2 {
		return nil, "", E_INVALID
	}

	docname := terms[n-2]
	key = terms[n-1]

	docs := make([]system.DocId, n-1)
	docs[0] = system.DocId(docname)
	for i := 1; i < n-1; i++ {
		docs[i] = system.DocId(strings.Join(terms[0:i], ".") + "." + docname)
	}
	return docs, key, nil
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

func (env *Environment) Get(key varKey) (v interface{}, found bool) {
	env.varslock.RLock()
	defer env.varslock.RUnlock()

	v, found = env.vars[key]
	return
}

// nil value not accepted.
func (env *Environment) Set(key varKey, v interface{}) (prev interface{}, e error) {
	if v == nil {
		return nil, E_INVALID
	}

	env.varslock.Lock()
	defer env.varslock.Unlock()

	prev, _ = env.vars[key]
	env.vars[key] = v
	return prev, nil
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

var defaultDirMode = os.FileMode(0755)

// Creates a new LSF environment in directory path.
// Path must be an absolute path.
// Sets state flag for env.
// Returns error if called on existing environemnt at path.
func CreateEnvironment(dir string, force bool) (string, error) {
	if !IsAbsPath(dir) {
		return "", E_RELATIVE_PATH
	}

	userHome := system.UserHome()
	isUserHome := userHome == dir
	if !isUserHome {
		// create user level .lsf environment if not existing
		if _, e := CreateEnvironment(userHome, false); e != nil && e != E_EXISTING_LSF {
			return "", e
		}
	}

	root := rootAt(dir)
	exists := exists(root)
	if exists && !force {
		return "", E_EXISTING_LSF
	}

	uid := HexShaDigest(dir) // unique id for this absolute path
	var resource string      // unique resource identifier is a path
	switch isUserHome {
	case true:
		resource = path.Join(userHome, ".lsf-init")
	default:
		resource = path.Join(userHome, RootDir, uid+".lsf-init")
	}
	lock, ok, e := system.LockResource(resource, "create new lsf port")
	if e != nil {
		return "", e
	}
	if !ok {
		return "", E_CONCURRENT
	}
	defer lock.Unlock()

	e = os.RemoveAll(root)
	if e != nil {
		return "", e
	}

	e = os.Mkdir(root, os.ModeDir|defaultDirMode)
	if e != nil {
		panic(e) // what could it be?
	}

	//	log.Printf("open registrar in %q", root)
	registrar, e := system.StartRegistry(root)
	if e != nil {
		return "", e
	}
	//	log.Printf("DEBUG using registrar %s", registrar)
	defer func() { registrar.Stop() <- struct{}{} }()

	docid := system.DocId("system")
	data := map[string][]byte{
		"create-time": []byte(time.Now().String()),
	}
	_, e = registrar.CreateDocument(docid, data)
	if e != nil {
		return "", e
	}

	return root, nil
}

func (env *Environment) Shutdown() error {

	if env == nil {
		panic("BUG - Environment.Shutdown: env is nil")
	}

	env.lock.Lock()
	defer env.lock.Unlock()

	if !env.bound {
		return E_ILLEGALSTATE
	}

	if registrar := env.registrar; registrar != nil {
		registrar.Stop() <- struct{}{}
		<-registrar.Done()
	}

	//	port, _ := env.Get(VarHomePort)
	//	log.Printf("DEBUG: Environment.Shutdown: %s", port.(*Port).Path())

	return nil
}
func (env *Environment) Initialize(dir string) (err error) {

	defer anomaly.Recover(&err)

	env.lock.Lock()
	defer env.lock.Unlock() // TODO: these need deadlines.

	if env.bound {
		//		log.Println("DEBUG: env.Initialize: already bound - ignore Initialize()")
		return nil
	}
	if dir == "" {
		return E_INVALID
	}

	if !IsAbsPath(dir) {
		dir = path.Join(Wd(), dir)
	}
	root := rootAt(dir)

	// check if exists
	if !exists(root) {
		return E_NOTEXISTING_LSF
	}

	env.bound = true
	port, e := schema.NewLocalPort(root)
	anomaly.PanicOnError(e, "Environment.Initialize:", "schema.NewLocalPort")

	env.Set(VarHomePort, port) // panics
	e = env.startRegistrar()
	anomaly.PanicOnError(e, "Environment.Initialize:", "env.startRegistrar")

	sysdoc := system.DocId("system")
	env.loadDocuments([]system.DocId{sysdoc})
	_, e = env.registrar.ReadDocument(sysdoc)
	anomaly.PanicOnError(e, "Environment.Initialize:", "env.ReadDocument")

	return nil
}

func (env *Environment) startRegistrar() error {
	if env.registrar != nil {
		return E_ILLEGALSTATE_REGISTRAR_RUNNING
	}

	port, found := env.Get(VarHomePort)
	if !found {
		return fmt.Errorf("BUG - env var %q not set", VarHomePort)
	}
	home := port.(*schema.Port).Path()
	//	log.Printf("open registrar in %q", home)
	registrar, e := system.StartRegistry(home)
	if e != nil {
		return e
	}
	env.registrar = registrar
	//	log.Printf("DEBUG using registrar %s", env.registrar)

	return nil
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
	anomaly.PanicOnError(e, "Environment.GetResourceIds:", restype, "BUG - directory is empty", dir.Name())

	resIds := make([]string, len(dirnames))
	i := 0
	for _, dirname := range dirnames {
		if dirname[0] != '.' {
			resIds[i] = dirname; i++
		}
	}
	return resIds
}
func (env *Environment) GetResourceDigests (restype string, verbose bool, encoder system.DocumentDigestFn) []string {

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

func justResourceId(env *Environment, restype string, resid string, encode system.DocumentDigestFn) string {
	return resid
}

func digestForResourceId(env *Environment, restype string, resid string, encode system.DocumentDigestFn) string {
	docid := system.DocId(fmt.Sprintf("%s.%s.%s", restype, resid, restype))
	doc, e := env.LoadDocument(docid)
	anomaly.PanicOnError(e, "BUG", "getResourceDigests:", "loadDocument", docid)
	anomaly.PanicOnTrue(doc == nil, "BUG", "getResourceDigests:", "loadDocument", docid)
	return encode(doc)
}
