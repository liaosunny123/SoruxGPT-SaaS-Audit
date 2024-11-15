#!/bin/bash

set -e

gf build main.go -a amd64 -s linux -p ./temp
gf docker main.go -p -t epicmo/soruxgpt-saas-audit:latest
now=$(date +"%Y%m%d%H%M%S")
# 以当前时间为版本号
docker tag epicmo/soruxgpt-saas-audit:latest epicmo/soruxgpt-saas-audit:$now
docker push epicmo/soruxgpt-saas-audit:$now
echo "release success" $now
