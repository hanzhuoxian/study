#include <stdio.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    int i, fd;
    struct stat statbuf;
    struct timespec time[2];
    for (i = 1; i < argc; i++)
    {
        if (stat(argv[i], &statbuf) < 0)
        {
            char err[100];
            sprintf(err, "%s", argv[i]);
            perror(err);
            continue;
        }
        if ((fd = open(argv[i], O_RDWR | O_TRUNC)) < 0)
        {
            char err[100];
            sprintf(err, "%s", argv[i]);
            perror(err);
            continue;
        }

        time[0] = statbuf.st_atimespec;
        time[1] = statbuf.st_mtimespec;
        if (futimens(fd, time) < 0)
        {
            char err[100];
            sprintf(err, "%s", argv[i]);
            perror(err);
            continue;
        }
        close(fd);
    }
    return 0;
}
