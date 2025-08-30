#! /bin/bash

# sudo dnf -y install gcc
# sudo dnf -y install pcre-devel zlib-devel openssl-devel

NGINX_VERSION=${NGINX_VERSION:=1.28.0}
NGINX_MODULE_DIR=ngx_aegis_module
AEGIS_VERSION=${AEGIS_VERSION:=0.0.0}

# Prepare environment
CWD=$(pwd)
rm -rf temp
mkdir temp
rm -rf build/${NGINX_MODULE_DIR}-$NGINX_VERSION
mkdir -p build/${NGINX_MODULE_DIR}-$NGINX_VERSION
rm -rf build/aegis
mkdir -p build/aegis

# Download and extract nginx
curl -o nginx-${NGINX_VERSION}.tar.gz -L https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz
tar -C temp -xf nginx-${NGINX_VERSION}.tar.gz
mkdir temp/nginx-${NGINX_VERSION}/${NGINX_MODULE_DIR}
cp ../ngx_aegis_module/src/* temp/nginx-${NGINX_VERSION}/${NGINX_MODULE_DIR}/

# Build ngx_aegis_module
cd temp/nginx-${NGINX_VERSION}
./configure --add-dynamic-module=./ngx_aegis_module --with-compat
make modules
cp objs/ngx_aegis_module.so ${CWD}/build/${NGINX_MODULE_DIR}-$NGINX_VERSION/

# Build aegis
cd $CWD/..
go build -o ${CWD}/build/aegis/aegis cmd/main.go

# Build archive
cd $CWD
RELEADE_DIR=build/aegis-$AEGIS_VERSION.$NGINX_VERSION
mkdir -p $RELEADE_DIR/usr/bin
mkdir -p $RELEADE_DIR/usr/lib64/modules
cp -r package/etc $RELEADE_DIR
cp build/aegis/aegis $RELEADE_DIR/usr/bin/aegis
cp build/ngx_aegis_module-$NGINX_VERSION/ngx_aegis_module.so $RELEADE_DIR/usr/lib64/modules/ngx_aegis_module.so

cd $RELEADE_DIR
tar -czf ../aegis_nginx_$NGINX_VERSION-$AEGIS_VERSION.tar.gz *

rm -rf ${CWD}/temp
