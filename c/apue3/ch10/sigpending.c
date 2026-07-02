#include <stdio.h>
#include <signal.h>
#include <errno.h>
#include <stdlib.h>
#include <unistd.h>

static void sig_quit(int);

int main(int argc, char const *argv[])
{
    sigset_t newmask,
        oldmask, pendmask;

    // 自定义退出信号
    if (signal(SIGQUIT, sig_quit) == SIG_ERR)
    {
        printf("cant set signal SIGQUIT");
        exit(1);
    }
    // 阻塞退出信号
    sigemptyset(&newmask);
    sigaddset(&newmask, SIGQUIT);
    if (sigprocmask(SIG_BLOCK, &newmask, &oldmask) < 0)
    {
        printf("\nsigprocmask failed");
        exit(1);
    }

    // 在休眠期间发生退出事件
    sleep(20);

    // 获取未绝信号
    if (sigpending(&pendmask) < 0)
    {
        printf("\nsigpending failed");
        exit(1);
    }

    // 打印未绝信号
    if (sigismember(&pendmask, SIGQUIT))
    {
        printf("\n SIGQUIT pending");
    }
    // 设置为原始屏蔽信号
    if (sigprocmask(SIG_SETMASK, &oldmask, NULL) < 0)
    {
        printf("\nsig_setmask failed");
        exit(1);
    }
    printf("\nsigprocmask unblock!");
    sleep(10);
    return 0;
}

static void sig_quit(int signo)
{
    printf("\ncaught SIGQUIT!");
    if (signal(SIGQUIT, SIG_DFL) == SIG_ERR)
    {
        printf("\ncant't reset SIGQUIT");
        exit(1);
    }
}