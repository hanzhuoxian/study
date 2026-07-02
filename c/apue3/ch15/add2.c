#include <stdio.h>
#include <unistd.h>

#define MAXLINE 1000

int main(int argc, char const *argv[])
{
    int n, int1, int2;
    char line[MAXLINE];
    while ((n = read(STDIN_FILENO, line, MAXLINE)) > 0)
    {
        line[n] = 0;
        if (scanf(line, "%d%d", &int1, &int2))
        {
            sprintf(line, "%d\n", int1 + int2);
            n = strlen(line);
            if (write(STDOUT_FILENO, line, n) != n)
            {
                printf("wreite error\n");
                exit(1);
            }
        }
        else
        {
            if (write(STDOUT_FILENO, "invalid args\n", 13) != 13)
            {

                printf("wreite error\n");
                exit(1);
            }
        }
    }

    return 0;
}
