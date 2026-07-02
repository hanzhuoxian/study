#include <unistd.h>

ssize_t readn(int fd, void *ptr, size_t nbytes)
{
    size_t nleft;
    ssize_t nread;

    nleft = nbytes;
    while (nleft > 0)
    {
        if ((nread = read(fd, ptr, nleft)) < 0)
        {
            if (nleft == nbytes) // 如果第一次就写失败了，相当于一个字节也没有写入
            {
                return -1;
            }
            else // 以前可能写入过数据，会返回以前写入的字节数
            {
                break;
            }
        }
        nleft -= nread;
        ptr = ptr + nread;
    }
    return (n - nleft);
}