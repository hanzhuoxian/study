#include <unistd.h>

ssize_t writen(int fd, void *ptr, size_t nbytes)
{
    size_t nleft;
    ssize_t nwrite;

    nleft = nbytes;
    while ((nwrite = write(fd, ptr, nleft)) < 0)
    {
        if (nleft == nbytes)
        {
            return -1;
        }
        else
        {
            break;
        }
        nleft -= nwrite;
        ptr = ptr + nwrite;
    }

    return (nbytes - nleft);
}