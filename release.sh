#!/bin/bash

now=$(date +"%Y%m%d%H%M%S")
docker buildx build --platform linux/amd64 -t epicmo/soruxgpt-saas-audit:latest --push .
docker buildx build --platform linux/amd64 -t epicmo/soruxgpt-saas-audit:latest --push .

echo "release success" $now
