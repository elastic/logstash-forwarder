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
	"math/big"
	"net"
	"os"
	"testing"
	"time"
)

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
}

func tryConnect(addr string) chan error {
	ch := make(chan error)
	go func() {

		caCertFile, err := ioutil.TempFile("", "logstash-forwarder-cacert")
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(caCertFile.Name(), []byte(caCert), os.ModeTemporary)
		cleanup := func() {
			os.Remove(caCertFile.Name())
		}

		// this can be messy because of localhost resolving to ipv6 addresses
		// but there's no easy way to diasble v6 resolution here
		const wait = 5
		timeout := time.AfterFunc(time.Second*wait, func() {
			ch <- errors.New("Client couldn't connect & handshake")
			cleanup()
		})

		sock := connect(&NetworkConfig{
			SSLCA:   caCertFile.Name(),
			Servers: []string{addr},
			Timeout: wait,
			timeout: time.Second * wait,
		})
		timeout.Stop() // cancel the timeout panic
		defer cleanup()
		if sock == nil {
			ch <- errors.New("connection should not be nil")
			return
		}
		if !sock.ConnectionState().HandshakeComplete {
			ch <- errors.New("handshake should be complete")
			return
		}
		defer sock.Close()
		ch <- nil
	}()
	return ch
}

// CA certificate is CN=ca.logstash.test in test/ca.crt, test/ca.key
// Server certificate is CN=localhost, signed by above CA, in test/server.crt, test/server.key
func TestConnectValidCertificate(t *testing.T) {
	go listenWithCert("localhost", "0.0.0.0:19876")
	if err := <-tryConnect("localhost:19876"); err != nil {
		t.Fatal("Should have succeeded", err)
	}
}

func TestConnectMismatchedCN(t *testing.T) {
	go listenWithCert("localalt", "0.0.0.0:19876")
	if err := <-tryConnect("localhost:19876"); err == nil {
		t.Fatal("Should have failed but didn't!")
	}
}

func TestConnectToIpWithoutSAN(t *testing.T) {
	go listenWithCert("localhost", "0.0.0.0:19876")
	if err := <-tryConnect("127.0.0.1:19876"); err == nil {
		t.Fatal("Should have failed but didn't!")
	}
}

func TestConnectToIpWithSAN(t *testing.T) {
	go listenWithCert("127.0.0.1", "0.0.0.0:19876")
	if err := <-tryConnect("127.0.0.1:19876"); err != nil {
		t.Fatal("Should not have failed", err)
	}
}
