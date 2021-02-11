#!/bin/bash
#
# build script to compile backwards compatible Mac version
#
CGO_CFLAGS="-mmacosx-version-min=10.12" CGO_LDFLAGS="-mmacosx-version-min=10.12" go build
