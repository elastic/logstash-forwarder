openssl genrsa -out server.key 1024
openssl req -new -key server.key -batch -out server.csr
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
