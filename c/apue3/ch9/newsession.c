#include <stdio.h>
#include <unistd.h>

static void pr_ids(char *name)
{
    printf("%s: pid=%ld, ppid=%ld, pgrp = %ld, tpgrp = %ld\n", name, (long)getpid(), (long)getppid(), (long)getpgrp(), (long)tcgetpgrp(STDIN_FILENO));
}

int main(int argc, const char* argv[]) {
    pid_t pid;
    if ((pid=fork())<0) {
        printf("fork error\n"); 
    } else if (pid > 0) {
        // parent
        pr_ids("parent");
        return 0;
    } else {
        //child
        pr_ids("child");
        setsid();
    }
    return 0;
}
