#include <signal.h>
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <stdlib.h>
#include "../ch10/pr_mask.h"

/**
 * 线程1
 **/
static void *th_fn1(void *arg)
{
    pr_mask((char *)arg);
    int err;
    sigset_t oldmask, mask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGINT);
    if ((err = pthread_sigmask(SIG_BLOCK, &mask, &oldmask)) != 0)
    {
        printf("SIG_BLOCK error\n");
        exit(err);
    }
    printf("thread 1 set sig_block success\n");
    pr_mask((char *)arg);
    return ((void *)0);
}

/**
 * 线程2
 **/
static void *th_fn2(void *arg)
{
    sleep(3);
    pr_mask((char *)arg);
    sigset_t mask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGINT);
    printf("thread 2 sigwait\n");
    int err = sigwait(&mask, NULL);
    if (err != 0)
    {
        printf("sigwait is failed");
    }
    printf("thread 2 sigwait done\n");
    pr_mask((char *)arg);
    return ((void *)0);
}

int main(int argc, char const *argv[])
{
    void *ret;
    int err;
    pthread_t tid1, tid2;
    sigset_t mask, oldmask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGINT);
    if ((err = pthread_sigmask(SIG_BLOCK, &mask, &oldmask)) != 0)
    {
        printf("SIG_BLOCK error\n");
        exit(err);
    }
    err = pthread_create(&tid1, NULL, th_fn1, "thread 1");
    if (err != 0)
    {
        printf("can't create thread 1\n");
    }
    err = pthread_create(&tid2, NULL, th_fn2, "thread 2");
    if (err != 0)
    {
        printf("can't create thread 2\n");
    }
    err = pthread_kill(tid1, 0);
    if (err == 0)
    {
        printf("tid1 is runing\n");
    }
    err = pthread_join(tid1, &ret);
    if (err != 0)
    {
        printf("can't join thread 1, errno : %d\n", err);
    }

    err = pthread_join(tid2, &ret);
    if (err != 0)
    {
        printf("can't join thread 2, errno : %d\n", err);
    }
    pr_mask("main after set sigmask");
    return 0;
}
