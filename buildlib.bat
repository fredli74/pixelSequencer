del libimagequant_win.a
cd libimagequant
gcc -w -c -O3 -fno-math-errno -funroll-loops -fomit-frame-pointer -Wall -Wno-attributes -std=c99 -DNDEBUG -DUSE_SSE=1 -msse -fexcess-precision=fast^
  pam.c mediancut.c blur.c mempool.c kmeans.c nearest.c libimagequant.c
ar rcs ../libimagequant_win.a *.o
cd ..