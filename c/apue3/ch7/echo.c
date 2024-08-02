#include <stdio.h>

const int MAX_NUM = 10;
const int MIX_NUM = 11;

int main(int argc, char const *argv[])
{
    int i;
    int j;
    printf("%p\n", &i);
    printf("%p\n", &j);
    printf("%p\n", &MAX_NUM);
    printf("%p\n", &MIX_NUM);
    for (i = 0; i < argc; i++)
    {
        printf("argv[%d]: %s\n", i, argv[i]);
    }

    for (i = 0; argv[i] != NULL; i++)
    {
        printf("argv[%d]: %s\n", i, argv[i]);
    }
    return 0;
}
