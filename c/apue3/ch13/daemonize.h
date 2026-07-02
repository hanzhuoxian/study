#include <stdio.h>
#include <syslog.h>
#include <sys/resource.h>
#include <unistd.h>
#include <signal.h>
#include <sys/stat.h>
#include <stdlib.h>
#include <fcntl.h>
// #include <apue.h>

void daemonize(const char *cmd)
{
    printf("daemon is started!\n");
    int i, fd0, fd1, fd2;
    pid_t pid;
    struct rlimit rl;
    struct sigaction sa;

    // clear file creation mask
    umask(0);

    // get maximum number of file descriptiors
    if (getrlimit(RLIMIT_NOFILE, &rl) < 0)
    {
        printf("%s can't get file limit", cmd);
        exit(1);
    }

    printf("max no file is %d", (int)rl.rlim_max);

    // become a session leader to lose controlling tty
    printf("pid: %d fork a process!\n", getpid());
    if ((pid = fork()) < 0)
    {
        printf("%s can't fork", cmd);
        exit(2);
    }
    else if (pid != 0) // parent
    {
        printf("parent 1 is end\n");
        exit(0);
    }
    printf("child %d is started!\n", getpid());

    setsid();

    sa.sa_handler = SIG_IGN;

    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    if (sigaction(SIGHUP, &sa, NULL) < 0)
    {
        printf("%s can't ignore SIGHUP", cmd);
        exit(3);
    }

    printf("child %d is fork a process!\n", getpid());
    if ((pid = fork()) < 0)
    {
        printf("%s can't fork", cmd);
        exit(3);
    }
    else if (pid != 0) // parent
    {
        printf("parent 2 is end\n");
        exit(0);
    }

    printf("child %d is start\n", getpid());

    printf("change work dir is /\n");
    // change the current working directory to the root so
    // we won't prevent file systems from being unmounted
    if (chdir("/") < 0)
    {
        printf("%s can't change director to /", cmd);
        exit(5);
    }
    printf("close fd all!\n");
    // close all open file descriptiors
    if (rl.rlim_max == RLIM_INFINITY)
    {
        rl.rlim_max = 1024;
    }
    for (i = 0; i < rl.rlim_max; i++)
    {
        close(i);
    }

    printf("close fd all sucess!\n");

    printf("set fd 0,1,2\n");
    // attach file descriptors 0, 1,and 2 to /dev/null
    fd0 = open("/dev/null", O_RDWR);
    fd1 = dup(0);
    fd2 = dup(0);

    printf("open log is start\n");
    // Initialize the log file
    openlog(cmd, LOG_CONS, LOG_DAEMON);
    if (fd0 != 0 || fd1 != 1 || fd2 != 2)
    {
        // syslog(LOG_ERR, "unexpected file descriptors %d %d %d", fd0, fd1, fd2);
        printf("unexpected file descriptors %d %d %d", fd0, fd1, fd2);
        exit(1);
    }
    printf("daemon end\n");
}

int main(int argc, char const *argv[])
{
    printf("daemon main start\n");
    daemonize("/opt/homebrew/bin/php");
    sleep(60);
    printf("daemon main finish\n");
    return 0;
}
