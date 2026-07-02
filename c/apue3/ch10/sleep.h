#include <stdio.h>
#include <signal.h>
#include <sys/signal.h>
#include <unistd.h>

static void sig_alrm(int signo)
{
    // 什么都不需要做，由于SIGALRM信号的默认动作是终止进程，设置空的信号处理程序去唤醒sigsuspend
}

unsigned int sleep(unsigned int seconds)
{
    struct sigaction newact, oldact;
    sigset_t newmask, oldmask, suspmask;
    unsigned int unslept;
    // 设置信号处理器
    // newact.sa_handler = sig_alrm;
    newact.sa_handler = SIG_IGN;
    sigemptyset(&newact.sa_mask);
    newact.sa_flags = 0;
    sigaction(SIGALRM, &newact, &oldact);

    // 阻塞SIGALRM并且保存当前信号阻塞值
    sigemptyset(&newmask);
    sigaddset(&newmask, SIGALRM);
    sigprocmask(SIG_BLOCK, &newmask, &oldmask);

    alarm(seconds);

    suspmask = oldmask;
    // 确保SIGALRM没有被阻塞
    sigdelset(&suspmask, SIGALRM);

    // 设置阻塞suspmak中的信号并将进程挂起
    sigsuspend(&suspmask);

    // 获取剩余时间
    unslept = alarm(0);

    // 恢复之前的信号处理器
    sigaction(SIGALRM, &oldact, NULL);
    // 恢复之前的信号阻塞
    sigprocmask(SIG_SETMASK, &oldmask, NULL);
    return unslept;
}