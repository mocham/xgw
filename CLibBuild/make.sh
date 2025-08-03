#! /usr/bin/bash
docker build -t make .
mkdir -p $PWD/CACHE/
docker run --rm -v "$PWD":"$PWD" --entrypoint /usr/bin/rsync make -r /output/static-libs $PWD/CACHE/
docker run --rm -v "$PWD":"$PWD" --entrypoint /usr/bin/rsync make -r /output/include $PWD/CACHE/
