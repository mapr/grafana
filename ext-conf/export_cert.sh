#!/bin/bash
# script to extract the MCS cert and keys to use for web services

MAPR_HOME="${MAPR_HOME:-/opt/mapr}"
MAPR_USER="${MAPR_USER:-mapr}"
MAPR_GROUP="${MAPR_GROUP:-mapr}"
sslKeyStore="/opt/mapr/conf/ssl_keystore"
tempKeystore="/tmp/keystore.$$.p12"
manageSSL="${MAPR_HOME}/server/manageSSLKeys.sh"
storeFormat=JKS
storePass="$(fgrep storePass $manageSSL | head -1 |cut -d'=' -f2 )"
CLUSTERNAME=$(cat /opt/mapr/conf/mapr-clusters.conf | awk '{print $1}' | head -n 1)
KEYTOOL=$(which keytool)

if [ -z "$KEYTOOL" ]; then
    echo "Failed to find keytool"
    exit 1
fi

if [ $# -ne 1 ]; then
    echo "Missing destination directory"
    echo "usage: $0 <key_dest_dir>"
    exit 1
fi

DEST_DIR=$1

if [ ! -d "$DEST_DIR" ]; then
    echo "$DEST_DIR is not a directory"
    exit 1
fi

if [ -z "$storePass" ]; then
    echo "Failed to extract keystore password"
    exit 1
fi

if [ -z "$CLUSTERNAME" ]; then
    echo "Failed to extract cluster name"
    exit 1
fi

$KEYTOOL -importkeystore -srckeystore $sslKeyStore -destkeystore $tempKeystore -deststoretype PKCS12 -srcalias $CLUSTERNAME  -srcstorepass $storePass -deststorepass $storePass #-destkeypass <password>

PW_FILE="/tmp/pw_ossl.tmp.$$"
# export cert:

openssl pkcs12 -in $tempKeystore -passin pass:$storePass -nokeys -out $DEST_DIR/cert.pem
RC1=$?

# export key:
openssl pkcs12 -in $tempKeystore -passin pass:$storePass -nodes -nocerts -out $DEST_DIR/key.pem
RC2=$?
if [ $RC2 -eq 0 ]; then
    chown ${MAPR_USER}:${MAPR_GROUP} $DEST_DIR/*.pem
    chmod 600 $DEST_DIR/key.pem
fi

rm -f $tempKeystore

if [ $RC1 -eq 0 -a $RC2 -eq 0 ]; then
    exit 0
else
    exit 1
fi

