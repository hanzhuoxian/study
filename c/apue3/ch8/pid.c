#include <stdio.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    printf("getpid %d\n", getpid());
    printf("getppid %d\n", getppid());
    printf("getuid %d\n", getuid());
    printf("geteuid %d\n", geteuid());
    printf("getgid %d\n", getgid());
    printf("getegid %d\n", getegid());
    return 0;
}
