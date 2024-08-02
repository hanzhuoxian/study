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

void sigterm(int signo)
{
    syslog(LOG_INFO, "got SIGTERM: exiting");
    exit(0);
}

void sighup(int signo)
{
    syslog(LOG_INFO, "Re-reading configuration file");
    exit(0);
}

int main(int argc, char const *argv[])
{
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

    // handle signals of interest
    sa.sa_handler = sigterm;
    sigemptyset(&sa.sa_mask);
    sigaddset(&sa.sa_mask, SIGHUP);
    sa.sa_flags = 0;
    if (sigaction(SIGTERM, &sa, NULL) < 0)
    {
        printf("can't restore SIGTERM default");
        exit(1);
    }
    // restore SIGHUP default and block all signals
    sa.sa_handler = sighup;
    sigemptyset(&sa.sa_mask);
    sigaddset(&sa.sa_mask, SIGTERM);
    sa.sa_flags = 0;
    if (sigaction(SIGHUP, &sa, NULL) < 0)
    {
        printf("can't restore SIGHUP default");
        exit(1);
    }
    return 0;
}
