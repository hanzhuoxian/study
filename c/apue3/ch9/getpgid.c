#include <unistd.h>
#include <stdio.h>

int main(int argc, const char *argv[])
{
    pid_t pgid = getpgid(0);
    if (pgid == -1)
    {
        printf("getpgid is error");
        return -1;
    }
    pid_t pgid1 = sepgid(getpid(), getpid());
    setsid();
    if (pgid1 == -1)
    {
        printf("setpgid is error");
        return -2;
    }
    printf("getpid is %d\n", getpid());
    printf("getppid is %d\n", getppid());
    printf("getpgid is %d\n", pgid);
    printf("getsid is %d\n", getsid(0));

    return 0;
}
