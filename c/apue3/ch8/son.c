#include <sys/wait.h>
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>

int main(int argc, char const *argv[])
{
    pid_t pid;
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0) // first child
    {
        printf("id first child %d\n", getpid());
        if ((pid = fork()) < 0)
        {
            printf("fork error\n");
        }
        else if (pid > 0) //第一个儿子
        {
            printf("id first child %d\n", getpid());
            exit(0); // parent from second fork == first child
        }
        // 第二个儿子
        printf("id second child %d\n", getpid());
        sleep(2);
        printf("second child, parent pid = %ld\n", (long)getppid());
        exit(0);
    }

    if (waitpid(pid, NULL, 0) != pid) // 父进程会等待子进程
    {
        printf("waitpid error\n");
    }

    return 0;
}
