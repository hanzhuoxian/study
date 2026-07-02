#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include "lock_reg.h"
#include "lock_test.h"
#include <errno.h>

void printInfo(char *name, struct flock lock)
{
    printf("%s pid %d whence:%d type:%d, start:%d len:%d\n", name, (int)getpid(), lock.l_whence, lock.l_type, (int)lock.l_start, (int)lock.l_len);
}

int main(int argc, char const *argv[])
{
    int fd;
    pid_t pid;
    int ret;
    // 打开文件
    if ((fd = open("../lock.lock", O_WRONLY | O_CREAT, S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP)) < 0)
    {
        perror(argv[0]);
    }
    else
    {
        printf("open filen lock.lock success\n");
    }

    if ((pid = fork()) < 0)
    {
        perror(argv[0]);
    }
    else if (pid == 0)
    {
        struct flock child_lock;
        child_lock.l_type = F_WRLCK;
        child_lock.l_whence = SEEK_SET;
        child_lock.l_start = 0;
        child_lock.l_len = 2;
        printInfo("child", child_lock);
        if ((ret = fcntl(fd, F_SETLKW, &child_lock)) < 0)
        {
            perror(argv[0]);
        }
        else
        {
            printf("write lock success\n");
        }
    }
    else
    {
        struct flock parent_lock;
        parent_lock.l_type = F_WRLCK;
        parent_lock.l_whence = SEEK_SET;
        parent_lock.l_start = 0;
        parent_lock.l_len = 2;
        printInfo("parent", parent_lock);
        if ((ret = fcntl(fd, F_SETLKW, &parent_lock)) < 0)
        {
            perror(argv[0]);
        }
        else
        {
            printf("write lock success\n");
        }
    }
    return 0;
}
