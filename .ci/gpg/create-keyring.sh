#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# Modified from https://blogs.itemis.com/en/secure-your-travis-ci-releases-part-2-signature-with-openpgp

function err_exit() {
  echo "ERROR: ${1:-"Unknown Error"} Exiting." 1>&2
  exit 1
}

declare -r GPG_HOME="${DIR}/keyring"
declare -r SECRING_AUTO="${GPG_HOME}/secring.auto"
declare -r PUBRING_AUTO="${GPG_HOME}/pubring.auto"

mkdir -p --mode 700 "$GPG_HOME"
cp "${DIR}"/*.auto* "${GPG_HOME}"

echo -e "\nImporting public keys..."
{ gpg --home "${GPG_HOME}" --import "${PUBRING_AUTO}" ; } || { err_exit "Could not import public key into gpg." ; }
echo "Success!"

echo -e "\nDecrypting secret key..."
{
  # $GPG_PASSWORD is taken from the script's env (injected by CI).
  echo $GPG_PASSWORD | gpg --home "${GPG_HOME}" --decrypt \
    --pinentry-mode loopback --batch \
    --passphrase-fd 0 \
    --output "${SECRING_AUTO}" \
    "${SECRING_AUTO}".gpg ; \
} || { err_exit "Failed to decrypt secret key." ; }
echo "Success!"

echo -e "\nImporting private keys..."
# { gpg --home "${GPG_HOME}" --import "${PUBRING_AUTO}" ; } || { err_exit "Could not import public key into gpg." ; }
{ gpg --home "${GPG_HOME}" --import "${SECRING_AUTO}" ; } || { err_exit "Could not import secret key into gpg." ; }
echo "Success!"
