#include <stdio.h>
#include <fcntl.h>
#include <unistd.h>

#define RWRWRW (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH)

int main(int argc, char const *argv[])
{
    // 新建文件
    int fd = open("./len.txt", O_CREAT | O_WRONLY, RWRWRW);
    if (fd < 0)
    {
        perror(argv[0]);
    }

    printf("len.txt fd: %d\n", fd);
    char hello[100] = "hello";

    int c = write(fd, &hello, sizeof(hello));
    if (c == -1)
    {
        perror("write failed");
    }

    off_t offset = 1000000;
    int loff_t = lseek(fd, offset, SEEK_END);
    if (loff_t == -1)
    {
        perror("set lseek failed");
    }

    int d = write(fd, &hello, sizeof(hello));
    if (d == -1)
    {
        perror("write failed");
    }

    return 0;
}
