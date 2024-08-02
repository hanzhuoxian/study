#include <stdio.h>

#include <ctype.h>

int main(int argc, char const *argv[])
{
    int c;

    while ((c = getchar()) != EOF)
    {
        if (isupper(c))
        {
            c = tolower(c);
        }

        if (putchar(c) == EOF)
        {
            printf("output error");
            exit(1);
        }

        if (c == "\n")
        {
            fflush(stdout);
        }
    }
    return 0;
}