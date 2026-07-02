#include <sys/wait.h>
#include <errno.h>
#include <signal.h>
#include <unistd.h>

int system(const char *cmdstring)
{
    pid_t pid;
    int status;
    struct sigaction ignore, saveintr, savequit;
    sigset_t chldmask, savemask;

    if (cmdstring == NULL)
    {
        return 1;
    }

    // ignore SIGINT and SIGQUIT
    ignore.sa_handler = SIG_IGN;
    sigemptyset(&ignore.sa_mask);
    ignore.sa_flags = 0;
    if (sigaction(SIGINT, &ignore, &saveintr) < 0)
    {
        return -1;
    }
    if (sigaction(SIGQUIT, &ignore, &savequit) < 0)
    {
        return -1;
    }

    // now block SIGCHLD
    sigemptyset(&chldmask);
    sigaddset(&chldmask, SIGCHLD);
    sigprocmask(SIG_BLOCK, &chldmask, &savemask);

    if ((pid = fork()) < 0)
    {
        status = -1; // probably out of processes
    }
    else if (pid == 0) // child
    {
        // restore previous signal actions & reset signal mask
        sigaction(SIGINT, &saveintr, NULL);
        sigaction(SIGQUIT, &savequit, NULL);
        sigprocmask(SIG_SETMASK, &savemask, NULL);
        execl("/bin/sh", "sh", "-c", cmdstring, (char *)0);
        _exit(127);
    }
    else // parent
    {
        while (waitpid(pid, &status, 0) < 0)
        {
            if (errno != EINTR)
            {
                status = -1;
                break;
            }
        }
    }

    // restore previous signal actions & reset signal mask
    if (sigaction(SIGINT, &saveintr, NULL) < 0)
    {
        return -1;
    }
    if (sigaction(SIGQUIT, &savequit, NULL) < 0)
    {
        return -1;
    }
    if (sigprocmask(SIG_SETMASK, &savemask, NULL) < 0)
    {
        return -1;
    }

    return status;
}