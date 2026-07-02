#include <unistd.h>
#include <stdio.h>
#include <fcntl.h>

#define RWRWRW (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH)

int main(int argc, char const *argv[])
{
    // open file
    int truncate_fd = open("./truncate.txt", O_CREAT | O_WRONLY, RWRWRW);
    char hello_txt[6] = "hello\n";
    size_t w_size = sizeof(hello_txt);
    int w;
    if ((w = write(truncate_fd, &hello_txt, w_size)) != w_size)
    {
        perror("write failed");
    }
    if ((w = truncate("./truncate.txt", 1)) < 0)
    {
        perror("truncate is failed");
    }
    if ((w = ftruncate(truncate_fd, 0)) < 0)
    {
        perror("truncate is failed");
    }
    close(truncate_fd);
    return 0;
}
