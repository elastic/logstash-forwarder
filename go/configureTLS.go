package main
import (
  "crypto/tls"
)

func configureTLS(capath string) (tlsconf *tls.Config, err error) {
  certpool, err := loadCertificates(capath)
  if (err != nil) { return }
  tlsconf = &tls.Config{RootCAs: certpool, InsecureSkipVerify: false}
  return
}
