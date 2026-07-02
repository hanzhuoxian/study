#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>

#define is_read_lockable(fd, offset, whence, len) (lock_test((fd), F_RDLCK, (offset), (whence), (len)) == 0)
#define is_write_lockable(fd, offset, whence, len) (lock_test((fd), F_WRLCK, (offset), (whence), (len)) == 0)

pid_t lock_test(int fd, int type, off_t offset, int whence, off_t len)
{
    struct flock lock;
    lock.l_type = type;
    lock.l_start = offset;
    lock.l_whence = whence;
    lock.l_len = len;
    if (fcntl(fd, F_GETLK, &lock) < 0)
    {
        printf("fcntl error");
        exit(1);
    }
    if (lock.l_type == F_UNLCK)
    {
        return 0; // false, region isn't locked by another proc
    }
    return lock.l_pid; // true, return pid of lock owner
}