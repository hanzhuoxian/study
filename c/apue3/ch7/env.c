#include <stdlib.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    char *name = getenv("USERNAME");
    if (name != NULL)
    {

        printf("%s\n", name);
    }
    char *columns = getenv("COLUMNS");
    if (columns != NULL)
    {
        printf("%s\n", columns);
    }
    setenv("HOME", "/home/madison1", 2);
    char *home = getenv("HOME");
    if (home != NULL)
    {
        printf("%s\n", home);
    }
    putenv("HOME1=/home/madison2");
    char *home1 = getenv("HOME1");
    if (home1 != NULL)
    {
        printf("%s\n", home1);
    }
    char *pwd = getenv("PWD");
    if (pwd != NULL)
    {
        printf("%s\n", pwd);
    }
    return 0;
}
