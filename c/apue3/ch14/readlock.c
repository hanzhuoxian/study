#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <string.h>

int main(int argc, char const *argv[])
{
    printf("main start\n");
    int fd = open("./lock.lock", O_WRONLY);
    if (fd < 0)
    {
        printf("open fd ret %d, errno %d : %s", fd, errno, strerror(errno));
    }
    struct flock lock;
    lock.l_type = F_WRLCK;
    lock.l_start = 0;
    lock.l_len = 0;
    lock.l_whence = SEEK_SET;
    int ret = fcntl(fd, F_SETLK, &lock);
    if (ret == -1)
    {
        printf("fcntl ret %d, errno %d : %s\n", ret, errno, strerror(errno));
    }
    printf("pid : %d", (int)getpid());
    pause();

    printf("main end\n");
    return 0;
}
