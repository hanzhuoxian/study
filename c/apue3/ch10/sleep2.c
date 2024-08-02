#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <setjmp.h>

// 定义jmpbuf
static sigjmp_buf jmpbuf;

static void sig_alarm(int signo)
{
    sigsetjmp(jmpbuf, 1);
}

unsigned int sleep(unsigned int seconds)
{
    if (signal(SIGALRM, sig_alarm) != SIG_ERR)
    {
        return seconds;
    }
    if (siglongjmp(jmpbuf) == 0)
    {
        alarm(seconds);
        pause();
    }
    return (alarm(0));
}

int main(int argc, char const *argv[])
{
    return 0;
}
