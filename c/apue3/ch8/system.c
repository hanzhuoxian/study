#include <sys/wait.h>
#include <errno.h>
#include <unistd.h>
#include <stdio.h>
#include "pr_exit.c"

int system(const char *);

int main(int argc, char const *argv[])
{
    int status;
    if ((status = system("date > date.log")) < 0)
    {
        printf("system() error");
    }
    pr_exit(status);

    if ((status = system("nosuchcommand")) < 0)
    {
        printf("system() error");
    }
    pr_exit(status);

    if ((status = system("who;exit 44")) < 0)
    {
        printf("system() error");
    }
    pr_exit(status);
    return 0;
}
