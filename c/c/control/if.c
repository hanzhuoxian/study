#include <stdio.h>

int main(int argc, char const *argv[])
{
    int i = 10;
    if (i > 5)
        printf("i>5\n");
    else if (i > 10)
        printf("i>10\n");
    else if (i > 20)
        printf("i>20\n");
    else
        printf("i<5\n");
    return 0;
}
