#include <stdio.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <strings.h>

// 每次一页地显示已产生的输出，调用分页程序就，我们希望从管道将输出直接输送至分页程序
#define MAXLINE 10000
#ifdef __APPLE__
#define DEF_PAGER "/usr/bin/more"
#else
#define DEF_PAGER "/bin/more"
#endif

int main(int argc, char const *argv[])
{
    char line[MAXLINE];
    FILE *fpin, *fpout;
    if (argc != 2)
    {
        printf("usage: ./pipepage pathname\n");
        exit(1);
    }

    if ((fpin = fopen(argv[1], "r")) == NULL)
    {
        printf("can't open %s", argv[1]);
        exit(1);
    }

    if ((fpout = popen(DEF_PAGER, "w")) == NULL)
    {
        printf("popen error\n");
        exit(1);
    }

    while (fgets(line, MAXLINE, fpin) != NULL)
    {
        if (fputs(line, fpout) == EOF)
        {
            printf("fputs error to pipe");
            exit(2);
        }
    }

    if (ferror(fpin))
    {
        printf("fgets error");
        exit(2);
    }

    if (pclose(fpout) == -1)
    {
        printf("pclose error");
        exit(3);
    }
    return 0;
}
