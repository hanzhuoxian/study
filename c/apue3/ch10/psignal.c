#include <signal.h>
#include <unistd.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    extern const char *const sys_siglist[NSIG];
    size_t i;
    for (i = 0; i < NSIG; i++)
    {
        printf("%ld=%s\n", i, sys_siglist[i]);
    }
    return 0;
}
