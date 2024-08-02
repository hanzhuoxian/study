#include <stdlib.h>
#include <apue.h>

static void charatatime(char *);

int main(int argc, char const *argv[])
{
    pid_t pid;
    TELL_WAIT(); //加同步
    if ((pid = fork()) < 0)
    {
        printf("fork error\n");
    }
    else if (pid == 0)
    {
        WAIT_PARENT();
        charatatime("output from child\n");
    }
    else
    {
        charatatime("output from parent\n");
        TELL_CHILD(pid);
    }

    return 0;
}

static void charatatime(char *str)
{
    char *ptr;
    int c;
    setbuf(stdout, NULL); // set unbuffered
    for (ptr = str; (c = *ptr++) != 0;)
    {
        putc(c, stdout);
    }
}