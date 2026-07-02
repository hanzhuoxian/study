#include <stdio.h>
#include <signal.h>
#include <unistd.h>

#define BUFFSIZE 1024

static void sig_tstp(int signo)
{
    sigset_t mask;

    sigemptyset(&mask);
    sigaddset(&mask, SIGTSTP);
    sigprocmask(SIG_BLOCK, &mask, NULL);

    signal(SIGTSTP, SIG_DFL);

    kill(getpid(), SIGTSTP);

    signal(SIGTSTP, sig_tstp);
}
