#include <stdio.h>

int main(int argc, char const *argv[])
{

    char *path = "./ferror.c";
    FILE *fp = fopen(path, "r");
    if (NULL == fp)
    {
        printf("open file error");
        return 1;
    }
    if (ferror(fp) != 0)
    {
        printf("open file error");
    }
    else
    {
        printf("open file no errno");
    }

    if (feof(fp) != 0)
    {
        printf("feof!");
    }
    else
    {
        printf("no feof");
    }
    return 0;
}
