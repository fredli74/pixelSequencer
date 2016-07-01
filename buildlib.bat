cd pngquant\lib
gcc -w -c *.c
ar rcs libimagequant.a *.o
cd ..\..