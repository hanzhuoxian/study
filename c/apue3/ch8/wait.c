#include <stdio.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>

void pr_exit(int);

int main(int argc, char const *argv[])
{
    pid_t pid;
    int status;
    printf("main pid: %d\n", getpid()); // 父进程id

    if ((pid = fork()) < 0) // fork后父子进程都会执行以下代码
    {
        printf("fork error");
    }
    else if (pid == 0)
    {
        printf("exit pid: %d\n", getpid());
        exit(7); //子进程结束
    }
    else
    {
        printf("parent pid: %d\n", getpid()); // 父进程输出后继续执行剩下的代码
    }

    if (wait(&status) != pid)
    {
        printf("wait error\n");
    }
    pr_exit(status); // 父进程输出子进程的状态

    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        printf("abort pid: %d\n", getpid());
        abort();
    }
    else
    {
        printf("parent pid: %d\n", getpid());
    }

    if (wait(&status) != pid)
    {
        printf("wait error\n");
    }
    pr_exit(status);

    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        status /= 0;
        printf("divied pid: %d\n", getpid());
    }
    else
    {
        printf("parent pid: %d\n", getpid());
    }

    if (wait(&status) != pid)
    {
        printf("wait error\n");
    }
    pr_exit(status);

    return 0;
}

void pr_exit(int status)
{
    if (WIFEXITED(status))
    {
        printf("normal termination, exit status=%d\n", WEXITSTATUS(status));
    }
    else if (WIFSIGNALED(status))
    {
        printf("abnormal termination, signal number = %d%s\n", WTERMSIG(status),
#ifdef WCOREDUMP
               WCOREDUMP(status) ? "(core file generated)" : "");
#else
               "");
#endif
    }
    else if (WIFSTOPPED(status))
    {
        printf("child stopped, signal number=%d\n", WSTOPSIG(status));
    }
}