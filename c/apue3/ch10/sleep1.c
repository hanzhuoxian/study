#include <stdio.h>
#include <sys/signal.h>
#include <unistd.h>
#include <signal.h>
#include <time.h>

// nothing to do, just return to wake up the pause
static void sig_alarm(int signo) {}
// define sleep function
unsigned int sleep1(unsigned int seconds);

int main(int argc, char const *argv[])
{
    sleep1(10);
}

unsigned int sleep1(unsigned int seconds)
{
    if (signal(SIGALRM, sig_alarm) == SIG_ERR)
    {
        return (seconds);
    }
    alarm(seconds);
    pause();
    return (alarm(0));
}
