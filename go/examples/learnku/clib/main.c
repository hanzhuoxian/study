#include <stdio.h>

#include "libconv.h"

int main(int argc, char const *argv[])
{
    GoString s = {"123r", 3};
    GoInt i = myatoi(s);
    printf("call go atoi %lld\n", i);
    return 0;
}
