#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>

#define MAXLINE 1000

int main(int argc, char const *argv[])
{
    int n;
    int fd[2];
    pid_t pid;
    char line[MAXLINE];

    // 创建管道
    if (pipe(fd) < 0)
    {
        printf("pipe error\n");
        exit(1);
    }

    // 创建子进程
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
        exit(1);
    } else if (pid > 0) {
        // 父进程
        close(fd[0]);
        write(fd[1], "hello world\n", 12);
    } else {
        // 子进程
        close(fd[1]);
        n = read(fd[0], line, MAXLINE);
        write (STDOUT_FILENO, line , n);
    }
    return 0;
}
