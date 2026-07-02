#include <stdio.h>
#include <envz.h>
#include <stdlib.h>

extern char **environ;
int main(int argc, char const *argv[])
{
    printf("%s\n", getenv("USERNAME"));

    char **envir = environ;
    while (*envir != NULL)
    {
        puts(*envir);
        envir++;
    }

    return 0;
}
