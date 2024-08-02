#include <stdlib.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    float *f = malloc(sizeof(float));
    *f = 1.1;
    printf("f = %f &%p\n", *f, f);
    float *d = malloc(sizeof(float));
    *d = 1.1;
    printf("f = %f &%p\n", *d, d);
    free(f);
    return 0;
}
