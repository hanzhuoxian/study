#include <stdio.h>
#include <unistd.h>

int globvar = 6; // external variable in initialized data

char buf[] = "a write to stdout\n";

int main(int argc, char const *argv[])
{
    int var;
    pid_t pid;

    var = 88;
    if (write(STDOUT_FILENO, buf, sizeof(buf) - 1) != sizeof(buf) - 1)
    {
        printf("write error");
    }
    printf("before fork\n");
    if ((pid = fork()) < 0)
    {
        printf("fork error");
    }
    else if (pid == 0)
    {
        globvar++;
        var++;
    }
    else
    {
        sleep(2);
    }

    printf("pid = %ld, ppid = %ld, glob = %d, var = %d\n", (long)getpid(), (long)getppid(), globvar, var);
    return 0;
}
