#include <errno.h>
#include <unistd.h>
#include <stdio.h>
#include <sys/signal.h>
#include <signal.h>

static void sig_hup(int signo)
{
    printf("SIGHUP received, pid = %ld\n", (long)getpid());
}

static void sig_hup_continue(int signo)
{
    printf("SIGCONT received, pid = %ld\n", (long)getpid());
}

static void pr_ids(char *name)
{
    printf("%s: pid=%ld, ppid=%ld, pgrp = %ld, tpgrp = %ld\n", name, (long)getpid(), (long)getppid(), (long)getpgrp(), (long)tcgetpgrp(STDIN_FILENO));
}

int main(int argc, const char *argv[])
{
    char c;
    pid_t pid;
    pr_ids("parent");
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid > 0)
    {             // parent
        sleep(5); // sleep to let child stop itself
    }
    else
    {
        pr_ids("child");
        signal(SIGHUP, sig_hup); // establish signal handler
        signal(SIGCONT, sig_hup_continue); // establish signal handler
        kill(getpid(), SIGTSTP); // stop ourself
        pr_ids("child");         // prints only if we're continued
        if (read(STDIN_FILENO, &c, 1) != 1)
        {
            printf("read error %d on controlling TTY\n", errno);
        }
    }
    return 0;
}
