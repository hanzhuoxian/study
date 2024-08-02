#include <stdio.h>

int main(int argc, char const *argv[])
{
    for (int i = 0; i < argc; i++)
    {
        printf("argv[%d]=%s\n", i, argv[i]);
    }

    char **ptr;
    extern char **environ;
    for (ptr = environ; *ptr != 0; ptr++)
    {
        printf("%s\n", *ptr);
    }
    return 0;
}
