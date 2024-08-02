#include <signal.h>
#include <stdio.h>
#include <errno.h>

void pr_mask(const char *str)
{
    sigset_t sigset;
    int errno_save;
    errno_save = errno;

    printf("program signo mask %s : ", str);
    if (sigprocmask(0, NULL, &sigset) < 0)
    {
        printf("sigprocmask errno\n");
    }
    else
    {
        if (sigismember(&sigset, SIGINT))
        {
            printf("SIGINT|");
        }
        if (sigismember(&sigset, SIGUSR1))
        {
            printf("SIGUSR1|");
        }
        if (sigismember(&sigset, SIGALRM))
        {
            printf("SIGALRM|");
        }
        printf("\n");
    }
    errno = errno_save; // restore errno
}