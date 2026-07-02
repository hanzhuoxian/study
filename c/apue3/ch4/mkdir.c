#include <unistd.h>
#include <fcntl.h>
#include <strings.h>
#include <stdio.h>
#include <sys/stat.h>

#define RWXRWXRW (S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IXGRP | S_IROTH | S_IWOTH)

int main(int argc, char const *argv[])
{
    if (mkdir("./dir", RWXRWXRW) < 0)
    {
        perror(argv[0]);
    }

    if (rmdir("./dir") < 0)
    {
        perror(argv[0]);
    }
    return 0;
}