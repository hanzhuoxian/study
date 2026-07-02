#include <stdio.h>
#include <setjmp.h>
#include <time.h>
#include <signal.h>
#include "pr_mask.h"
#include <unistd.h>
#include <stdlib.h>

// 定义usr1信号处理程序
static void sig_usr1(int);
// 定义alarm信号处理程序
static void sig_alrm(int);
// 定义jmpbuf
static sigjmp_buf jmpbuf;
// 是否能跳转
static volatile sig_atomic_t canjmp;

int main(int argc, char const *argv[])
{
    sigset_t newmask, oldmask;
    sigemptyset(&newmask);
    sigaddset(&newmask, SIGINT);

    if (sigprocmask(SIG_BLOCK, &newmask, &oldmask) < 0)
    {
        printf("block sigint failed\n");
        exit(2);
    }

    if (sigprocmask(SIG_SETMASK, &oldmask, NULL) < 0)
    {
        printf("set mask sigint failed\n");
        exit(3);
    }
    // 如果在这个时间发生了信号，那么进程将永远阻塞
    pause();

    if (signal(SIGUSR1, sig_usr1) == SIG_ERR)
    {
        printf("signal(SIGUSR1) error\n");
        exit(0);
    }

    if (signal(SIGALRM, sig_alrm) == SIG_ERR)
    {
        printf("signal(SIGALRM) error\n");
        exit(0);
    }

    pr_mask("starting main:");

    if (sigsetjmp(jmpbuf, 1))
    {
        pr_mask("ending main:");
        exit(0);
    }

    canjmp = 1;

    for (;;)
        pause();

    return 0;
}

static void sig_usr1(int signo)
{
    time_t starttime;

    if (canjmp == 0)
    {
        return; // unexpected signal, ignore
    }

    pr_mask("starting sig_usr1: ");

    alarm(3);

    starttime = time(NULL);

    for (;;)
        if (time(NULL) > starttime + 5)
            break;

    pr_mask("finishing sig_usr1");

    canjmp = 0;

    siglongjmp(jmpbuf, 1); // jmp back to main, don't return
}

static void sig_alrm(int signo)
{
    pr_mask("in sig_alrm:");
}