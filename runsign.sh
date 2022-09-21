#! /bin/sh

if [ $# -ne 2 ]
then
    echo runsign.sh port-device path-to-message
    exit 1
fi

./runapp --port $1 --file signerapp/app.bin
./tk1sign --port $1 --file $2

