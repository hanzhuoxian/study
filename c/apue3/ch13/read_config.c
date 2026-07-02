#include <stdio.h>
#include <unistd.h>
#include <pthread.h>
#include <syslog.h>
#include <signal.h>
#include <sys/signal.h>
#include "./daemonize.h"
// #include <apue.h>

sigset_t mask;

extern int already_runing(void);

void reread(void)
{
    printf("reread finished!");
}

void *thr_fn(void *arg)
{
    int err, signo;
    for (;;)
    {
        err = sigwait(&mask, &signo);
        if (err != 0)
        {
            syslog(LOG_ERR, "sigwait failed!");
            exit(1);
        }
        switch (signo)
        {
        case SIGHUP:
            syslog(LOG_INFO, "Re-reading configuration file");
            reread();
            break;
        case SIGTERM:
            syslog(LOG_INFO, "got sigterm; exiting");
            exit(0);
        default:
            syslog(LOG_INFO, "unexpected signal %d\n", signo);
            break;
        }
    }
    return 0;
}

int main(int argc, char const *argv[])
{
    int err;
    pthread_t tid;
    char *cmd;
    struct sigaction sa;

    if ((cmd = strrchr(argv[0], '/')) == NULL)
    {
        cmd = argv[0];
    }
    else
    {
        cmd++;
    }

    // become a daemon
    daemonize(cmd);

    if (already_runing())
    {
        syslog(LOG_ERR, "daemon already runing");
        exit(1);
    }

    // restore SIGHUP default and block all signals
    sa.sa_handler = SIG_DFL;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    if (sigaction(SIGHUP, &sa, NULL) < 0)
    {
        printf("can't restore SIGHUP default");
        exit(1);
    }
    sigfillset(&mask);
    if ((err = pthread_sigmask(SIG_BLOCK, &mask, NULL)) != 0)
    {
        printf("can't sig_block ERROR");
        exit(1);
    }
    err = pthread_create(&tid, NULL, thr_fn, 0);
    if (err != 0)
    {
        syslog(LOG_ERR, "can't create thread");
        exit(1);
    }
    return 0;
}
