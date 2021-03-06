#!/bin/bash

pushd () {
    command pushd "$@" > /dev/null
}

popd () {
    command popd > /dev/null
}


if [ "$#" -ne 2 ]; then
  echo "Usage: $0 SRCDIRNAME DISTNAME" >&2
  exit 1
fi

SRCNAME="$1"
DSTNAME="$2"

SRC="dist/${SRCNAME}"
DST="dist/${DSTNAME}"

if [ ! -d "${SRC}" ]; then
    echo "Source directory \"${SRC}\" does not exist"
    exit 1
fi

# Remove target if it exists
rm -rf "${DST}"
# Setup target
cp -r "${SRC}" "${DST}"

# echo "Packaging distribution"
# Copy dist files
cp LICENSE "${DST}"
cp README.md "${DST}"
cp pkg/config.example.yml "${DST}"

if [[ "${DST}" != *"windows"* ]]; then
    cp -R pkg/services "${DST}"
    # echo "Compressing tarbell"
    pushd dist || exit
    rm -f "${DSTNAME}.tar.gz"
	tar --exclude=".*" --owner=0 --group=0 -zcf "${DSTNAME}.tar.gz" "${DSTNAME}"
    popd
else
    # echo "Compressing zip file"
    pushd dist || exit
    rm -f "${DSTNAME}.zip"
    zip --exclude .\* -qr "${DSTNAME}.zip" "${DSTNAME}"
    popd
fi

rm -rf ${DST}
