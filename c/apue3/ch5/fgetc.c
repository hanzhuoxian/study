#include <stdio.h>

int main(int argc, char const *argv[])
{
    char *path = "./getc.c";
    FILE *fp = fopen(path, "r");
    if (fp == NULL)
    {
        printf("open %s\n", path);
        return 1;
    }

    int c;
    while ((c = f(fp)) != EOF)
    {
        printf("%c", c);
    }
    int success = fclose(fp);
    if (success != 0)
    {
        printf("close file failed");
        return 2;
    }
    return 0;
}
