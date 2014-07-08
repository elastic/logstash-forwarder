package schema

import (
	"fmt"
	"github.com/elasticsearch/kriterium/panics"
	"lsf/system"
	"net/url"
)

type PortId string
type portType int

// REVU: keep it as a memento .. who knows?
//const (
//	localPort  portType = 0
//	remotePort          = 1
//)

// REVU: do we still need this?
const AnonPortId PortId = ""

// ----------------------------------------------------------------------
// Port
// ----------------------------------------------------------------------

// lsf.RemotePort describes a remote LSF port.
type Port struct {
	local   bool
	Id      PortId // REVU: this needs a decision
	Host    string // double duty as local path
	PortNum string
	address *url.URL

	// todo certs ..
}

func (p Port) Address() string {
	return p.address.Path
}

// recorded elements of LogStream object
var PortElem = struct {
	Local, Id, Host, PortNum string
}{
	Local:   "local",
	Id:      "id",
	Host:    "host",
	PortNum: "port",
}

// REVU: TODO sort mappings at sysrec..
// NOTE: port-num image is expected to be a string
func (t *Port) Mappings() map[string][]byte {
	m := make(map[string][]byte)
	m[PortElem.Id] = []byte(t.Id)
	m[PortElem.Host] = []byte(t.Host)
	m[PortElem.PortNum] = []byte(t.PortNum)
	return m
}

func (t *Port) String() string {
	var locality = "local"
	if !t.local {
		locality = "remote"
	}
	return fmt.Sprintf("port %s %s %s", t.Id, locality, t.Address())
}

func PortDigest(doc system.Document) string {
	return DecodePort(doc).String()
}

func DecodePort(data system.DataMap) *Port {
	m := data.Mappings()

	host := string(m[PortElem.Host])
	portnumStr := string(m[PortElem.PortNum])

	var isLocal bool
	var canonicalPath string
	if len(portnumStr) > 0 {
		canonicalPath = fmt.Sprintf("%s:%s", host, portnumStr)
	} else {
		canonicalPath = host
		isLocal = true
	}
	addr, e := url.Parse(canonicalPath)
	panics.OnError(e, "BUG", "schema.DecodePort")

	id := string(m[PortElem.Id])
	port := &Port{
		local:   isLocal,
		Id:      PortId(id),
		Host:    host,
		PortNum: portnumStr,
		address: addr,
	}

	return port
}

// returns nil, nil on "" path input
// REVU: needs ID (REVU:/later: why? instead of portnum?)
func NewLocalPort(path string) (*Port, error) {
	if path == "" {
		return nil, nil
	}

	address, e := url.Parse(path)
	if e != nil {
		return nil, e
	}

	port := &Port{
		local:   true,
		Host:    path,
		address: address,
	}
	return port, nil
}

func NewRemotePort(id, host string, portno int) (*Port, error) {
	portnumStr := fmt.Sprintf("%d", portno)
	path := fmt.Sprintf("%s:%s", host, portnumStr)
	address, e := url.Parse(path)
	if e != nil {
		return nil, e
	}
	port := &Port{
		local:   false,
		Id:      PortId(id),
		Host:    host,
		PortNum: portnumStr,
		address: address,
	}
	return port, nil
}
