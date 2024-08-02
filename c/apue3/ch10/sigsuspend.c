#include <stdio.h>
#include <signal.h>
#include <unistd.h>

#include "pr_mask.h"

static void sig_int(int);

int main(int argc, char const *argv[])
{
    sigset_t newmask, oldmask, waitmask;
    pr_mask("program start: ");
    if (signal(SIGINT, sig_int) == SIG_ERR)
    {
        printf(" set sigint failed!\n");
        return 1;
    }
    sigemptyset(&waitmask);
    sigaddset(&waitmask, SIGUSR1);
    sigemptyset(&newmask);
    sigaddset(&newmask, SIGINT);

    // 阻塞SIGINT信号
    if (sigprocmask(SIG_BLOCK, &newmask, &oldmask) < 0)
    {
        printf("set SIG_BLOCK failed!");
        return 1;
    }

    pr_mask("in critical region: ");

    // 阻塞 SIGUSR1信号
    if (sigsuspend(&waitmask) != -1)
    {
        printf("sigsuspend error\n");
        return -1;
    }

    pr_mask("after return from sigsuspend: ");
    if (sigprocmask(SIG_SETMASK, &oldmask, NULL) < 0)
    {
        printf("sigprocmask setmask error\n");
        return -1;
    }

    pr_mask("program exit: ");
    return 0;
}

static void sig_int(int signo)
{
    pr_mask("\nin sig_int: ");
}