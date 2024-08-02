#include <stdio.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <strings.h>

// 每次一页地显示已产生的输出，调用分页程序就，我们希望从管道将输出直接输送至分页程序
#define MAXLINE 10000
#ifdef __APPLE__
#define DEF_PAGER "${PAGER:-more}"
#else
#define DEF_PAGER "/bin/more"
#endif

int main(int argc, char const *argv[])
{
    int n;
    int fd[2];
    pid_t pid;
    char *pager, *argv0;
    char line[MAXLINE];
    FILE *fp;

    if (argc != 2)
    {
        printf("usage: ./pipepage pathname\n");
        exit(1);
    }

    if ((fp = fopen(argv[1], "r")) == NULL)
    {
        printf("can't open %s", argv[1]);
        exit(1);
    }
    if (pipe(fd) < 0)
    {
        printf("pipe error");
        exit(1);
    }
    if ((pid = fork()) < 0)
    {
        printf("fork error");
        exit(1);
    }
    else if (pid > 0)
    {
        // parent
        close(fd[0]); // close read end
        while (fgets(line, MAXLINE, fp) != NULL)
        {
            n = strlen(line);
            if (write(fd[1], line, n) != n)
            {
                printf("write pipe error");
                exit(1);
            }
        }
        if (ferror(fp))
        {
            printf("fgets error");
        }

        close(fd[1]); // close write end of pipe for reader

        if (waitpid(pid, NULL, 0) < 0)
        {
            printf("waitpid error");
        }
        return 0;
    }
    else
    {
        close(fd[1]);
        if (fd[0] != STDIN_FILENO)
        {
            if (dup2(fd[0], STDIN_FILENO) != STDIN_FILENO)
            {
                printf("dup2 error to stdin");
            }
        }
        close(fd[0]);
        if ((pager = getenv("PAGER")) == NULL)
        {
            pager = DEF_PAGER;
        }
        pager = DEF_PAGER;
        if ((argv0 = strrchr(pager, '/')) != NULL)
        {
            argv0++;
        }
        else
        {
            argv0 = pager;
        }

        if (execl(pager, argv0, (char *)0) < 0)
        {
            printf("execl error for %s", pager);
        }
    }
    return 0;
}
