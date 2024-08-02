#include <stdio.h>
#include <unistd.h>
#include <errno.h>
#include <fcntl.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include "proc.h"
#include "lock_reg.h"

int main(int argc, char const *argv[])
{
    int fd;
    pid_t pid;
    char buf[5];
    struct stat statbuf;

    if (argc != 2)
    {
        fprintf(stderr, "usage: %s filename \n", argv[0]);
        exit(1);
    }

    // 创建文件
    if ((fd = open(argv[1], O_CREAT | O_RDWR | O_TRUNC, S_IRWXU | S_IRWXG)) < 0)
    {
        printf("open error\n");
        exit(1);
    }
    // 写入数据
    if (write(fd, "abcdef", 6) != 6)
    {
        printf("write error\n");
        exit(2);
    }
    // 查看文件状态
    if (fstat(fd, &statbuf) < 0)
    {
        printf("fstat error\n");
        exit(3);
    }
    // 设置执行组ID 关闭组执行位
    if (fchmod(fd, (statbuf.st_mode & ~S_IXGRP) | S_ISGID) < 0)
    {
        printf("fchmod error\n");
        exit(4);
    }

    TELL_WAIT();

    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
        exit(1);
    }
    else if (pid > 0)
    { // parent
        printf("parent pid %d\n", (int)getpid());
        if (write_lock(fd, 0, SEEK_SET, 0) < 0)
        {
            printf("write lock error\n");
            exit(1);
        }
        TELL_CHILD(pid);
        if (waitpid(pid, NULL, 0) < 0)
        {
            printf("waitpid error\n");
        }
    }
    else
    { // child
        printf("child pid %d\n", (int)getpid());
        WAIT_PARENT();

        fcntl(fd, F_SETFL, O_NONBLOCK);

        if (read_lock(fd, 0, SEEK_SET, 0) != -1)
        {
            printf("child: read_lock succeeded\n");
        }
        printf("read_lock of already-locked region returns %d\n", errno);
        if (lseek(fd, 0, SEEK_SET) == -1)
        {
            printf("lseek error");
        }
        if (read(fd, buf, 2) < 0)
        {
            printf("read failed (mandatory locking works)");
        }
        else
        {
            printf("read ok (no mandatory locking), buf = %2.2s\n", buf);
        }
    }
    return 0;
}
