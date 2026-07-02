#include <stdio.h>
#include "sleep.h"

int main(int argc, char const *argv[])
{
    unsigned int s = sleep(20);
    printf("%d", s);
    return 0;
}
