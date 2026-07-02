#include <stdio.h>
#include <signal.h>
#include <errno.h>
#include <stdlib.h>
#include <unistd.h>

void pr_mask(const char *str)
{
    sigset_t sigset;
    int errno_save;

    errno_save = errno;
    printf("program signo mask : %s\n", str);
    if (sigprocmask(0, NULL, &sigset) < 0)
    {
        printf("sigprocmask errno");
    }
    else
    {
        if (sigismember(&sigset, SIGINT))
        {
            printf("SIGINT\n");
        }
    }
    errno = errno_save; // restore errno
}

int main(int argc, char const *argv[])
{
    printf("This program is disable SIGINT signo\n");
    sigset_t set;
    sigemptyset(&set);
    sigaddset(&set, SIGINT);
    sigprocmask(SIG_BLOCK, &set, NULL);
    pr_mask("main");
    pause();
    return 0;
}
