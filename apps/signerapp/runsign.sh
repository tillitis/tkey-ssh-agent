#! /bin/sh

if [ $# -lt 1 ]; then
  cat <<EOF
Usage: runsign.sh path-to-message [common tkey-runapp and tkey-sign flags]

runsign.sh is a helper script that uses tkey-runapp to load the signerapp onto
TKey and start it. It then uses tkey-sign to request a signature of the
contents of the provided file (message). If --port or --speed flags needs to be
passed to tkey-runapp and tkey-sign, they can be passed after the message
argument.
EOF
  exit 2
fi

if [ ! -e "$1" ] || [ ! -f "$1" ]; then
  printf "Please give a path to an existing file as first argument\n"
  exit 2
fi

msgf="$1"
shift

./tkey-runapp --file apps/signerapp/app.bin "$@"
./tkey-sign --file "$msgf" "$@"
