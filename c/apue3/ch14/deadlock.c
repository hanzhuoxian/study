#include <fcntl.h>
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include "lock_reg.h"
#include "proc.h"
#include <errno.h>

static void lockabyte(const char *name, int fd, off_t offset)
{
    if (writew_lock(fd, offset, SEEK_SET, 1) < 0)
    {
        printf("%s : writew_lock error %s\n", name, strerror(errno));
        exit(1);
    }
    printf("%s : got the lock, byte %lld\n", name, (long long)offset);
}

int main(int argc, char const *argv[])
{
    int fd;
    pid_t pid;

    // 创建一个文件并且写两个字节
    if ((fd = creat("templock", S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH)) < 0)
    {
        printf("creat error\n");
        exit(1);
    }
    if (write(fd, "ab", 2) != 2)
    {
        printf("write error\n");
        exit(1);
    }
    TELL_WAIT();
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
        exit(1);
    }
    else if (pid == 0) // child
    {
        lockabyte("child", fd, 0);
        printf("child pid : %d\n", (int)getpid());
        printf("parent pid : %d\n", (int)getppid());
        TELL_PARENT(getppid());
        WAIT_PARENT();
        lockabyte("child", fd, 1);
    }
    else // parent
    {
        lockabyte("parent", fd, 1);
        TELL_CHILD(pid);
        WAIT_CHILD();
        lockabyte("parent", fd, 0);
    }

    return 0;
}
