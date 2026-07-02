#include <unistd.h>
#include <fcntl.h>
#include <strings.h>
#include <stdio.h>
#include <pwd.h>

#define RWRWRW (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH)

int main(int argc, char const *argv[])
{
    int chown_fd = open("./chown.txt", O_CREAT, RWRWRW);
    if (chown("./chown.txt", 501, 20) < 0)
    {
        perror("chown chown.txt");
    }
    if (fchown(chown_fd, 501, 20))
    {
        perror("fchown chown.txt");
    }
    close(chown_fd);
    return 0;
}
