#!/bin/bash

set -ex

rm -rf *.o *.so *.c *.so.* lib2

cat <<'EOF' >> lib.c
void assign_i(int *i)
{
   *i=5;
}
EOF

cat <<'EOF' >> bin.c
#include <stdio.h>
void assign_i(int *);

void _start()
{
   int x;
   assign_i(&x);
   asm("mov $60,%rax; mov $10,%rdi; syscall");
}
EOF

gcc -Wall -fPIC -c lib.c
gcc -shared -Wl,-soname,libsomething.so -o libsomething.so *.o
gcc -Wall -nostdlib -L./ bin.c -lsomething -o bin

mkdir lib2
gcc -shared -Wl,-soname,libsoho.so -o lib2/libsoho.so lib.o
gcc -Wall -Wl,-rpath,'$ORIGIN/lib2' -nostdlib -L./lib2 bin.c -lsoho -o bin2


rm -rf *.c *.o lib2
cp /lib64/ld-linux-x86-64.so.2 ld.so.2
