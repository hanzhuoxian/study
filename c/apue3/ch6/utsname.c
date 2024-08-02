#include <sys/utsname.h>
#include <stdio.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    char hostname[64];
    int hs = gethostname(hostname, sizeof(hostname));
    if (hs == 0)
    {
        printf("gethostname: %s\n", hostname);
    }

    struct utsname name;
    int s = uname(&name);
    if (s < 0)
    {
        perror("uname error:");
        return -1;
    }
    printf("uname.sysname: %s\n", name.sysname);
    printf("uname.nodename: %s\n", name.nodename);
    printf("uname.release: %s\n", name.release);
    printf("uname.machine: %s\n", name.machine);
    printf("uname.version: %s\n", name.version);

    return 0;
}
