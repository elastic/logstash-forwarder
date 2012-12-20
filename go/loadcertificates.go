package main

import (
  "crypto/x509"
  "io/ioutil"
  "encoding/pem"
)

func loadCertificates(path string) (certpool *x509.CertPool, err error) {
  pemblock, err := ioutil.ReadFile(path)
  if err != nil {
    return
  }

  certpool = x509.NewCertPool()

  for {
    derblock, x := pem.Decode(pemblock)
    pemblock = x
    if derblock == nil {
      break
    }

    cert, err := x509.ParseCertificate(derblock.Bytes)
    if err == nil {
      certpool.AddCert(cert)
    }
  }
  return
} /* loadCertificates */
