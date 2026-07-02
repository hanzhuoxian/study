#include <stdio.h>
#include <sys/wait.h>

int main(int argc, char const *argv[])
{
    char line[1000];
    FILE *fpin;

    if ((fpin = popen("myuclc", 'r')) == NULL)
    {
        printf("popen error\n");
        exit(1);
    }

    for (;;)
    {
        fputs("prompt> ", stdout);
        fflush(stdout);

        if (fgets(line, 1000, fpin) == NULL)
        {
            break;
        }
        if (fputs(line, stdout) == EOF)
        {
            printf("fputs error to pipe");
        }
    }

    if (pclose(fpin) == -1)
    {
        printf("pclose error\n");
    }

    putchar("\n");
    return 0;
}
