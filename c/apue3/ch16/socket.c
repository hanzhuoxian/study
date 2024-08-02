#include <stdio.h>
#include <sys/socket.h>
#include <unistd.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>

int main(int argc, char const *argv[])
{
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd == -1)
    {
        printf("create socket failed\n");
        exit(1);
    }
    printf("fd = %d\n", fd);
    int succ = shutdown(fd, SHUT_RDWR);
    if (succ != 0)
    {
        printf("close socket failed %d %d %s\n", succ, errno, strerror(errno));
        exit(1);
    }
    printf("succ = %d\n", succ);
    return 0;
}
