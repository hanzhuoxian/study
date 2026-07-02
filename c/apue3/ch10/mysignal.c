#include <signal.h>
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>

typedef void SigFunc(int);

SigFunc *mysignal(int signo, SigFunc *func)
{
    struct sigaction act, oact;
    act.sa_handler = func;
    sigemptyset(&act.sa_mask);
    act.sa_flags = 0;
    if (sigaction(signo, &act, &oact) < 0)
    {
        return SIG_ERR;
    }
    return (oact.sa_handler);
}

static void sig_int(int signo)
{
    printf("sig_int\n");
    sleep(5);
    printf("sig int end\n");
    exit(1);
}

static void sig_fpe(int signo)
{
    printf("sig_fpe\n");
    printf("sig fpe end\n");
    exit(2);
}

static void sig_term(int signo)
{
    printf("sig_term\n");
    printf("sig_term end\n");
    exit(3);
}

int main(int argc, char const *argv[])
{
    printf("main start\n");
    if (mysignal(SIGINT, sig_int) == SIG_ERR)
    {
        printf("set mysignal failed\n");
        exit(1);
    }

    if (mysignal(SIGFPE, sig_fpe) == SIG_ERR)
    {
        printf("set mysignal failed\n");
        exit(1);
    }

    if (mysignal(SIGTERM, sig_term) == SIG_ERR)
    {
        printf("set mysignal failed\n");
        exit(1);
    }

    sleep(20);
    return 0;
}
