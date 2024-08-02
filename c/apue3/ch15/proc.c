#include <stdio.h>
#include <unistd.h>
#include <signal.h>
#include <sys/signal.h>

// 创建了两个管道，父进程、子进程 各自关闭它们 不需要使用的管道端。必须使用两个管道，一个用作协同进程的标准输入，一个用作它的标准输出
// 子进程调用dup2,将管道描述符移至标准输入和标准输出，最后调用了execl
#define MAXLINE 1000

static void sig_pipe(int);

int main(int argc, char const *argv[])
{
    int n, fd1[2], fd2[2];
    pid_t pid;
    char line[MAXLINE];

    if (SIGNAL(SIGPIPE, sig_pipe) == SIG_ERR)
    {
        printf("signal error\n");
        exit(1);
    }

    if (pipe(fd1) < 0 || pipe(fd2) < 0)
    {
        printf("pipe error\n");
        exit(1);
    }

    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
        exit(1);
    }
    else if (pid > 0)
    {
        // parent
        close(fd1[0]);
        close(fd2[1]);

        while (fgets(line, MAXLINE, stdin) != NULL)
        {
            n = strlen(line);
            if (write(fd1[1], line, n) != n)
            {
                printf("write error\n");
                exit(1);
            }

            if ((n = read(fd2[0], line, MAXLINE)) < 0)
            {
                printf("read error from pipe");
            }

            if (n == 0)
            {
                printf("child closed pipe");
                break;
            }

            line[n] = 0; // null terminate
            if (fputs(line, stdout) == EOF)
            {
                printf("fputs error\n");
                exit(1);
            }
        }
        if (ferror(stdin))
        {
            printf("fgets error on stdin\n");
            exit(1);
        }
        return 0;
    }
    else
    {
        close(fd1[1]);
        close(fd2[0]);

        if (fd1[0] != STDIN_FILENO)
        {
            if (dup2(fd1[0], STDIN_FILENO) != STDIN_FILENO)
            {
                printf("dup2 error to stdin");
                exit(1);
            }

            close(fd1[0]);
        }

        if (execl("./add2", "add2", (char *)0) < 0)
        {
            printf("execl error\n");
            exit(1);
        }
    }

    return 0;
}

static void sig_pipe(int signo)
{
    printf("SIGPIPE caught\n");
    exit(1);
}