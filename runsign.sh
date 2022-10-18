#! /bin/sh

if [ $# -ne 2 ] && [ $# -ne 3 ]; then
  printf "runsign.sh port-device [port-speed] path-to-message\n"
  exit 2
fi

if [ $# -eq 2 ]; then
  ./runapp  --port "$1" --file apps/signerapp/app.bin
  ./tk1sign --port "$1" --file "$2"
else
  ./runapp  --port "$1" --speed "$2" --file apps/signerapp/app.bin
  ./tk1sign --port "$1" --speed "$2" --file "$3"
fi
