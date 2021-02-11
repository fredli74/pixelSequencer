#!/bin/bash
cd libimagequant
# ./configure
/bin/bash configure
make static
mv libimagequant.a ../libimagequant_unix.a
