#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <stdlib.h>

static volatile sig_atomic_t sigflag;
static sigset_t newmask, oldmask, zeromask;

// one signal handler for SIGUSR1 and SIGUSR2
static void sig_usr(int signo)
{
    sigflag = 1;
    printf("pid %d set sigflag \n", (int)getpid());
}

void TELL_WAIT(void)
{
    if (signal(SIGUSR1, sig_usr) == SIG_ERR)
    {
        printf("signal(SIGUSR1) error\n");
        exit(1);
    }
    if (signal(SIGUSR2, sig_usr) == SIG_ERR)
    {
        printf("signal(SIGUSR2) error\n");
        exit(1);
    }

    sigemptyset(&zeromask);
    sigemptyset(&newmask);
    sigaddset(&newmask, SIGUSR1);
    sigaddset(&newmask, SIGUSR2);
    if (sigprocmask(SIG_BLOCK, &newmask, &oldmask) < 0)
    {
        printf("SIG_BLOCK error");
        exit(1);
    }
    printf("TELL_WAAIT\n");
}

void TELL_PARENT(pid_t pid)
{
    printf("TELL_PARENT\n");
    kill(pid, SIGUSR2); // tell parent we're done
}

void WAIT_PARENT(void)
{
    printf("WAIT_PARENT\n");
    while (sigflag == 0)
    {
        sigsuspend(&zeromask); // and wait for parent
        printf("WAIT_PARENT sigsuspend\n");
    }
    printf("WAIT_PARENT SUCCESS\n");
    sigflag = 0;
    if (sigprocmask(SIG_SETMASK, &oldmask, NULL) < 0)
    {
        printf("SIG_SETMASK error\n");
    }
}

void TELL_CHILD(pid_t pid)
{
    printf("TELL_CHILD\n");
    kill(pid, SIGUSR1);
}

void WAIT_CHILD(void)
{
    printf("WAIT_CHILD\n");
    while (sigflag == 0)
    {
        printf("WAIT_CHILD before sigsuspend\n");
        sigsuspend(&zeromask); // and wait for child
        printf("WAIT_CHILD sigsuspend\n");
    }
    sigflag = 0;
    printf("WAIT_CHILD SUCCESS\n");

    if (sigprocmask(SIG_SETMASK, &oldmask, NULL) < 0)
    {
        printf("SIG_SETMASK error\n");
    }
}