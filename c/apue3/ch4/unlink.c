#include <unistd.h>
#include <fcntl.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    if (open("tempfile", O_CREAT | O_RDWR) < 0)
    {
        perror("open or creat tempfile failed");
        return 1;
    }
    if (unlink("tempfile") < 0)
    {
        perror("unlink error");
        return 2;
    }
    printf("file unlinked\n");
    sleep(15);
    printf("done\n");
    return 0;
}
