#include <stdio.h>

// #include "../ch8/system.h"
#include "./system.h"

// 定义SIGINT信号处理程序
static void sig_int(int signo)
{
    printf("caught SIGINT\n");
}

// 定义SIGCHLD信号处理程序
static void sig_chld(int signo)
{
    printf("caught SIGCHLD\n");
}

// 使用第8章中的system函数
int main(int argc, char const *argv[])
{
    if (signal(SIGINT, sig_int) == SIG_ERR)
    {
        printf("SIGINT error\n");
        return 1;
    }
    if (signal(SIGCHLD, sig_chld) == SIG_ERR)
    {
        printf("SIGCHLD error\n");
        return 2;
    }
    if (system("/bin/ed") < 0)
    {
        printf("system() error\n");
    }

    return 0;
}
