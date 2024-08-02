#include <stdio.h>
#include "external.h"

int main(int argc, char const *argv[])
{
    extern int age;
    printf("age = %d\n", age);
    printf("name = %s\n", NAME);
    return 0;
}
