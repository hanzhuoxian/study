#include <unistd.h>
#include <stdio.h>
#include <sys/wait.h>
#include <stdlib.h>

char *env_init[] = {"USER=unknown", "PATH=/Users/dxm/work/cstudy/apue3/ch8/", NULL};

int main(int argc, char const *argv[])
{
    putenv("PATH=/Users/dxm/work/cstudy/apue3/ch8/:.");
    pid_t pid;
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        printf("first child\n");
        if ((execle("/Users/dxm/work/cstudy/apue3/ch8/echoall.app", "echoall", "myarg1", "MY ARG2", (char *)0, env_init) < 0))
        {
            printf("execle error");
        }
    }
    sleep(2);
    if (waitpid(pid, NULL, 0) < 0)
    {
        printf("wait error");
    }
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        printf("first child child\n");
        if (execlp("echoall.app", "echoall", "only 1 arg", (char *)0) < 0)
        {
            printf("execlp error\n");
        }
    }

    return 0;
}
