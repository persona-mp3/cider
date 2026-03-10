#!/usr/bin/env bash

# I've not implemented the tls-protocol 
# before, but I know enough of how it works
# This was gotten from: https://gist.github.com/denji/12b3a568f092ab951456

# generates private key
openssl genrsa -out server.key 2048


# Key considerations for algorithm "ECDSA" ≥ secp384r1
# List ECDSA the supported curves (openssl ecparam -list_curves)
# openssl ecparam -genkey -name secp384r1 -out server.key

# # Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)
# openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650


openssl ecparam -genkey -name secp384r1 -out server.key

openssl req -new -x509 -sha256 \
  -key server.key \
  -out server.crt \
  -days 3650 \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
