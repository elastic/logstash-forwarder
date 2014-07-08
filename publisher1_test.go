package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const strict bool = true

const insecure bool = false

const caKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDXQK+POFCtDlYgc2nQTnZ+WfaPQg1ms9JjomZ9vZXpqH9JaxBj
jWKTyg7k6GOXpNbaET76nlWLtAKKofesOGwadil/HtyEwXxvXJY/UTdkqtwCFjPS
5XR5fNTkVK2sLw23Z67TIBApIGR42a7WhaP/BFv6zc/wpxDnokpLtJkvlQIDAQAB
AoGAA6paUvoGhavk05CjkKSFaAloJXBegg012/R8AlAkKWJxKHLmSIuzzgh20HcU
mxR3hCcfB22Cz2o1UN8JNKmRTaoMrPHf4gv0MIlcEBumxh8nyFiBocXimqJKHWHY
PMWzOoyhgBIXPoAIkmo9Ft41LidJ3FBl0z74muGcYsdu4FECQQD1nwfXHBP5jE2X
vVc5SupIxIgoK9reCGB2CyYdQtkdRPTO7bSLwTTqFlzjLYNaM3xZhG6Qh/tHIrE0
95MuumIDAkEA4Fkl8yVj+Pkx7gAGcEQoRwupk6gE/FM0WTJrpSc9+thNLk5DYCod
qwxmju8ttfr6wrIE1vDfK6njVo1a+RqAhwJABNmFABxP0KeSiKJ2bG0sPw+SWKi1
A5lKvknuELnXK5rG8qcC35eLAew7HUkyxL8rf2D8BeKJdZgbw533y/5mHQJBAJXL
MEmOl5evWyUcIzBmcbYuFUWfk+Sd8X/06GbXMs0AC1h1rQrSVAjXOMsK66xsLW49
ynlxTrEqt74cl7dneJUCQQDFwBdpxWZtEeIx0uliUZNoAUX5D8qA4/BuHgstXREw
5rWQly3kCyFmocbv7WggnNnmgIk3V2P9Vj3n8ZFLCYOJ
-----END RSA PRIVATE KEY-----`

const caCert = `-----BEGIN CERTIFICATE-----
MIICRTCCAa4CCQC/GQitAOqHJTANBgkqhkiG9w0BAQUFADBnMQswCQYDVQQGEwJB
VTERMA8GA1UECBMIVmljdG9yaWExEjAQBgNVBAcTCU1lbGJvdXJuZTEWMBQGA1UE
ChMNRWxhc3RpY1NlYXJjaDEZMBcGA1UEAxMQY2EubG9nc3Rhc2gudGVzdDAeFw0x
NDA3MDQwMTIwMjNaFw0yNDA3MDEwMTIwMjNaMGcxCzAJBgNVBAYTAkFVMREwDwYD
VQQIEwhWaWN0b3JpYTESMBAGA1UEBxMJTWVsYm91cm5lMRYwFAYDVQQKEw1FbGFz
dGljU2VhcmNoMRkwFwYDVQQDExBjYS5sb2dzdGFzaC50ZXN0MIGfMA0GCSqGSIb3
DQEBAQUAA4GNADCBiQKBgQDXQK+POFCtDlYgc2nQTnZ+WfaPQg1ms9JjomZ9vZXp
qH9JaxBjjWKTyg7k6GOXpNbaET76nlWLtAKKofesOGwadil/HtyEwXxvXJY/UTdk
qtwCFjPS5XR5fNTkVK2sLw23Z67TIBApIGR42a7WhaP/BFv6zc/wpxDnokpLtJkv
lQIDAQABMA0GCSqGSIb3DQEBBQUAA4GBAFzkH8T+dU40g330QnDp2qO0XTfhNOsC
fjUOGYo7F6eqfBcQColcE+BLKc1aKEAAEvzokQi72L7xuOenJUzpGaIJXGkmGZsV
2OIO5Zf4ChZTMuut9yPjer9sTt0pZUNsOSg6o7hBeXlCMEvoM/31ag2sxZaOKA/Z
p/X0O4Qz0RTF
-----END CERTIFICATE-----`

var listening sync.WaitGroup

func init() { log.SetFlags(0) }

func makeCert(host string) tls.Certificate {
	ca, err := tls.X509KeyPair([]byte(caCert), []byte(caKey))
	caCert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		panic(err)
	}
	tpl := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(0),
		Subject:               pkix.Name{CommonName: host},
		NotBefore:             time.Now().AddDate(-1, 0, 0).UTC(),
		NotAfter:              time.Now().AddDate(1, 0, 0).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{1, 2, 3, 4},
		Version:               2,
	}
	if ip := net.ParseIP(host); ip != nil {
		tpl.IPAddresses = []net.IP{ip}
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, caCert, &key.PublicKey, ca.PrivateKey)
	if err != nil {
		panic(err)
	}
	bcrt := &pem.Block{Type: "CERTIFICATE", Bytes: der}
	bkey := &pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}

	v, err := tls.X509KeyPair(pem.EncodeToMemory(bcrt), pem.EncodeToMemory(bkey))
	if err != nil {
		panic(err)
	}
	return v
}

func listenWithCert(hostname string, address string) {

	listening.Add(1)
	go func() {
		log.Println("DEBUG - start mock server ..")
		// Establish a dummy TLS server
		var serverConfig tls.Config
		kp := makeCert(hostname)

		serverConfig.Certificates = []tls.Certificate{kp}

		listener, err := tls.Listen("tcp", address, &serverConfig)
		if err != nil {
			panic(err)
		}
		// Listen and handshake for a single connection
		defer listener.Close()
		listening.Done()

		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		tlsconn, ok := conn.(*tls.Conn)
		if !ok {
			panic("conn should of *tls.Conn")
		}
		if err := tlsconn.Handshake(); err != nil {
			return
		}
	}()
	listening.Wait()
}

func tryConnect(addr string, strict bool) (errchan chan error) {
	errchan = make(chan error)
	go func() {

		caCertFile, err := ioutil.TempFile("", "logstash-forwarder-cacert")
		if err != nil {
			panic(err)
		}
		defer func() { os.Remove(caCertFile.Name()) }()
		ioutil.WriteFile(caCertFile.Name(), []byte(caCert), os.ModeTemporary)

		// this can be messy because of localhost resolving to ipv6 addresses
		// but there's no easy way to disable v6 resolution here
		const wait = 5
		const retryLimit = 3
		tryAttempt := 0
		exinfo := ""
		config := &NetworkConfig{
		SSLCA:     caCertFile.Name(),
		Servers:   []string{addr},
		Timeout:   wait,
		timeout:   time.Second * wait,
	}

		var socket *tls.Conn
		for socket == nil && tryAttempt < retryLimit {
			select {
			case socket = <-doConnect(config):
			case <-time.After(time.Second * wait):
				log.Printf("INFO: Connect timeout: attempt: %d\n", tryAttempt)
				tryAttempt++
			}
		}
		if socket == nil {
			errchan <- errors.New("Client connect failed. " + exinfo)
			return
		}
		defer socket.Close()
		log.Printf("INFO: Connected to %s\n", socket.RemoteAddr())

		if !socket.ConnectionState().HandshakeComplete {
			errchan <- errors.New("handshake should be complete")
			return
		}
		errchan <- nil
	}()
	return errchan
}

func doConnect(config *NetworkConfig) <-chan *tls.Conn {
	sockchan := make(chan *tls.Conn)
	go func() {
		sockchan <- connect(config)
	}()
	return sockchan
}

// ----------------------------------------------------------------------
// Strict
// ----------------------------------------------------------------------

// CA certificate is CN=ca.logstash.test in test/ca.crt, test/ca.key
// Server certificate is CN=localhost, signed by above CA, in test/server.crt, test/server.key

func TestStrictConnectValidCertificate(t *testing.T) {
	log.Println("\n-- TestStrictConnectValidCertificate -- ")

	listenWithCert("localhost", "0.0.0.0:19876")
	if err := <-tryConnect("localhost:19876", strict); err != nil {
		t.Fatal("Should have succeeded", err)
	}
}
func TestStrictConnectMismatchedCN(t *testing.T) {
	log.Println("\n-- TestStrictConnectMismatchedCN -- ")

	listenWithCert("localalt", "0.0.0.0:19876")
	if err := <-tryConnect("localhost:19876", strict); err == nil {
		t.Fatal("Should have failed but didn't!")
	}
}

func TestStrictConnectToIpWithoutSAN(t *testing.T) {
	log.Println("\n-- TestStrictConnectToIpWithoutSAN -- ")

	listenWithCert("localhost", "0.0.0.0:19876")
	if err := <-tryConnect("127.0.0.1:19876", strict); err == nil {
		t.Fatal("Should have failed but didn't!")
	}
}

func TestStrictConnectToIpWithSAN(t *testing.T) {
	log.Println("\n-- TestStrictConnectToIpWithSAN -- ")

	listenWithCert("127.0.0.1", "0.0.0.0:19876")
	if err := <-tryConnect("127.0.0.1:19876", strict); err != nil {
		t.Fatal("Should not have failed", err)
	}
}
