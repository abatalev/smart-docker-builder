#!/bin/sh

go build -o sdb .

./sdb -help
./sdb -version

for i in $(ls examples/Dockerfile.*);
do
    echo "==> test $i"
    ./sdb $i
done