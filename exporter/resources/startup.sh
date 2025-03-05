#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

echo "                                     ./////,                    "
echo "                                 ./////==//////*                "
echo "                                ////.  ___   ////.              "
echo "                         ,**,. ////  ,////A,  */// ,**,.        "
echo "                    ,/////////////*  */////*  *////////////A    "
echo "                   ////'        \VA.   '|'   .///'       '///*  "
echo "                  *///  .*///*,         |         .*//*,   ///* "
echo "                  (///  (//////)**--_./////_----*//////)   ///) "
echo "                   V///   '°°°°      (/////)      °°°°'   ////  "
echo "                    V/////(////////\. '°°°' ./////////(///(/'   "
echo "                       'V/(/////////////////////////////V'      "

echo "exporter started"

cp /root/.ssh/id_rsa.pub /root/.ssh/authorized_keys
chown -R root:root /root/.ssh
chmod -R 700 /root
chmod -R 600 /root/.ssh/*

mkdir -p /data/cas || true
mount -t cifs //samba-exporter-cas/Data /data/cas -o username=root,password=root,vers=3.0,rw || true

mkdir -p /data/mysql || true
mount -t cifs //samba-exporter-mysql/Data /data/mysql -o username=admin,password=admin,vers=3.0,rw,modefromsid || true

mkdir -p /data/ldap || true
mount -t cifs //samba-exporter-ldap/Data /data/ldap -o username=root,password=root,vers=3.0,rw || true

/usr/sbin/sshd -e -D