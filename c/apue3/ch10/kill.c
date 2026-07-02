#include <stdio.h>
#include <sys/signal.h>
#include <unistd.h>
#include <signal.h>

static void sig_usr(int);

int main(int argc, char const * argv[]) {
    if(signal(SIGUSR1,sig_usr) == SIG_ERR) {
        printf("can't catch SIGUSR1");
    }
    printf("kill send usr1\n");
    kill(getpid(), SIGUSR1);
    raise(SIGUSR1);
    for(;;) {
        pause();
    }
    return 0;
}

static void sig_usr(int signo)
{
    printf("received signal : %d\n",signo);
}
