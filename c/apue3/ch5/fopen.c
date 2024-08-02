#include <stdio.h>
#include <fcntl.h>

int main(int argc, char const *argv[])
{
    FILE *f = fopen("./fwide.c", "r");
    if (f == NULL)
    {
        printf("f is null");
    }
    printf("f is success\n");
    fclose(f);

    FILE *wf = freopen("./fopen.log", "w+", stdout);
    fputs("hello ", stdout);
    fputs("world !\n", wf);

    int fd = open("fopen.log", O_WRONLY);
    if (fd < 0)
    {
        printf("open fopen.log failed");
    }
    FILE *df = fdopen(fd, "a");
    fputs("hello fd\n", df);

    return 0;
}
