#include <stdio.h>
#include <apue.h>

int main(int argc, char const *argv[])
{
    char buf[MAXLINE];
    while (fgets(buf, MAXLINE, stdin) != NULL)
    {
        if (fputs(buf, stdout) == EOF)
        {
            perror("output error");
            exit(1);
        }
    }

    if (ferror(stdin))
    {
        perror("input error");
        exit(1);
    }

    return 0;
}
