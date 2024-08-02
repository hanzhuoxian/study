#include <sys/stat.h>
#include <stdio.h>
#include <fcntl.h>

int main(int argc, char const *argv[])
{
    int fd = open("./futimes.txt", O_CREAT, S_IRUSR | S_IWUSR);
    if (fd < 0)
    {
        perror(argv[0]);
        return 1;
    }

    // 空指针设置，时间为当前时间
    if (futimens(fd, NULL) < 0)
    {
        perror(argv[0]);
    }
    // UTIME_NOW 设置当前时间
    struct timespec now_times[2] = {{1627571055, UTIME_NOW}, {1627571055, UTIME_NOW}};
    if (futimens(fd, NULL) < 0)
    {
        perror(argv[0]);
    }

    // UTIME_NOW 设置当前时间
    struct timespec omit_times[2] = {{1627571055, UTIME_OMIT}, {1627571055, UTIME_OMIT}};
    if (futimens(fd, NULL) < 0)
    {
        perror(argv[0]);
    }
    // 指定相应的时间
    struct timespec times[2] = {{1627571055, 1111}, {1627571055, 22222}};
    if (futimens(fd, NULL) < 0)
    {
        perror(argv[0]);
    }

    return 0;
}
