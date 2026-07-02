#include <sys/wait.h>
#include <stdio.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    pid_t pid;
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        if (execl("/Users/dxm/work/cstudy/apue3/ch8/testinterp",
                  "testinterp", "myarg1", "MY ARG2", (char *)0) < 0)
        {
            printf("EXECL ERROR\n");
        }
    }
    if (waitpid(pid, NULL, 0) < 0)
    {
        printf("waitpid error\n");
    }
    return 0;
}