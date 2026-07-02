#include <stdio.h>
#include <errno.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>

#define BUFF_SIZE 5000000

char buf[BUFF_SIZE];

const int BLOCK = 1;
const int NO_BLOCK = 0;

int main(int argc, char const *argv[])
{
    int ntowrite, nwrite;
    char *ptr;

    ntowrite = read(STDIN_FILENO, buf, sizeof(buf));
    fprintf(stderr, "read %d bytes\n", ntowrite);

    ioctl(STDOUT_FILENO, FIONBIO, NO_BLOCK);

    ptr = buf;
    while (ntowrite > 0)
    {
        errno = 0;
        nwrite = write(STDOUT_FILENO, ptr, ntowrite);
        fprintf(stderr, "\nnwirte = %d, errno = %d \n", nwrite, errno);
        if (nwrite > 0)
        {
            ptr += nwrite;
            ntowrite -= nwrite;
        }
    }
    ioctl(STDOUT_FILENO, FIONBIO, BLOCK);
    return 0;
}
