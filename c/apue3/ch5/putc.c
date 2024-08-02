#include <stdio.h>

int main(int argc, char const *argv[])
{
    char *path = "./putc.log";
    FILE *fp = fopen(path, "w");
    if (fp == NULL)
    {
        printf("open %s\n", path);
        return 1;
    }
    putc('h', fp);
    fputc('e', fp);
    putchar('!');
    putchar('\n');
    return 0;
}
