#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

static int pfd1[2], pfd2[2];

void TELL_WAIT()
{
    if (pipe(pfd1) < 0 || pipe(pfd2) < 0)
    {
        printf("pipe error\n");
        exit(1);
    }
}

void TELL_PARENT(pid_t pid)
{
    if (write(pfd2[1], "c", 1) != 1)
    {
        printf("TELL_PARENT: write pipe error\n");
        exit(1);
    }
}

void WAIT_PARENT(pid_t pid)
{
    char c;
    if (read(pfd1[0], &c, 1) != 1)
    {
        printf("read pipe error\n");
        exit(1);
    }
    if (c != 'p')
    {
        printf("WAIT_PARENT: incorrect data\n");
        exit(1);
    }
}

void TELL_CHILD(pid_t pid)
{
    if (write(pfd1[1], "p", 1) != 1)
    {
        printf("TELL_CHILD: write pipe error\n");
        exit(1);
    }
}

void WAIT_CHILD(pid_t pid)
{
    char c;
    if (read(pfd2[0], &c, 1) != 1)
    {
        printf("read pipe error\n");
        exit(1);
    }
    if (c != 'c')
    {
        printf("WAIT_PARENT: incorrect data\n");
        exit(1);
    }
}