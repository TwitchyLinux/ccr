#!/bin/bash

set -ex

rm -rf *.o *.so *.c *.so.*

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

rm -rf *.c *.o
