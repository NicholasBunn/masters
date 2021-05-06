# This script generates the CA key and certificate, server key and certifcate, and signs the server's CSR.
# The generated files are used for TSL message encryption between client and server

# To get an understanding of self-signing SSL certificates (or at least, the arguements used below), 
# check out: https://linuxize.com/post/creating-a-self-signed-ssl-certificate/

rm *.pem

# Generate CA's private key and self-signed certificate
openssl req -newkey rsa:4096 \
            -x509 \
            -sha256 \
            -days 365 \
            -nodes \
            -out ca-cert.pem \
            -keyout ca-key.pem \
            -subj "/C=ZA/ST=Western Cape/L=Stellenbosch/O=Stelllenbosch University/OU=Engineering/CN=localhost/emailAddress=nicholasbunn@sun.ac.za"

echo "CA's self-signed certificate"
openssl x509 -in ca-cert.pem \
        -noout -text

# Generate server's private key and certificate signing request (CSR)
openssl req -newkey rsa:4096 \
            -nodes \
            -keyout server-key.pem \
            -out server-req.pem \
            -subj "/C=AQ/ST=Queen Maud Land/L=Vesleskarvet/O=SANAP/OU=Ship/CN=localhost/emailAddress=nicholasbunn@sun.ac.za"

# Use CA's private key to sign server's CSR and get back the signed certificate
openssl x509 -req \
        -in server-req.pem \
        -days 365 \
        -CA ca-cert.pem \
        -CAkey ca-key.pem \
        -CAcreateserial \
        -out server-cert.pem \
        -extfile server-ext.cnf

echo "Server's signed certificate"
openssl x509 -in server-cert.pem \
        -noout \
        -text

# Generate client's private key and certificate signing request (CSR)
openssl req -newkey rsa:4096 \
            -nodes \
            -keyout client-key.pem \
            -out client-req.pem \
            -subj "/C=AQ/ST=Queen Maud Land/L=Vesleskarvet/O=SANAP/OU=Ship/CN=localhost/emailAddress=nicholasbunn@sun.ac.za"

# Use CA's private key to sign client's CSR and get back the signed certificate
openssl x509 -req \
        -in client-req.pem \
        -days 365 \
        -CA ca-cert.pem \
        -CAkey ca-key.pem \
        -CAcreateserial \
        -out client-cert.pem \
        -extfile client-ext.cnf

echo "Client's signed certificate"
openssl x509 -in client-cert.pem \
        -noout \
        -text