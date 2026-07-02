
#include <stdio.h>
#include <sys/signal.h>
#include <unistd.h>
#include <signal.h>

static void sig_alarm(int);

int main(int argc, char const * argv[]) {
    if(signal(SIGALRM,sig_alarm) == SIG_ERR) {
        printf("can't catch alarm");
    }
    printf("kill send alarm\n");
    alarm(10);
    for(;;) {
        pause();
    }
    return 0;
}

static void sig_alarm(int signo)
{
    printf("received alarm signal no is : %d\n",signo);
}
