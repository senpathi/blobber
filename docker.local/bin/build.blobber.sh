#!/bin/sh

for i in $(seq 1 1);
do
  BLOBBER=$i docker-compose -p blobber$i -f docker.local/docker-compose.yml build --force-rm
done