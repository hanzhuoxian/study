#include <stdio.h>
#include <fcntl.h>
#include <unistd.h>

#define RWRWRW (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH)
int main(int argc, char const *argv[])
{
    if (open("./rename.txt", O_CREAT, RWRWRW) < 0)
    {
        perror(argv[0]);
        return 1;
    }
    if (rename("./rename.txt", "./rename.csv") < 0)
    {
        perror(argv[0]);
        return 2;
    }
    if (unlink("./rename.csv") < 0)
    {
        perror(argv[0]);
        return 3;
    }
    printf("rename rename.txt to rename.csv\n");
    return 0;
}
