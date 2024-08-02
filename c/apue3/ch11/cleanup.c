#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>

// 线程清理函数
void cleanup(void *arg);

// 线程1函数
void *thr_fn1(void *arg);
void *thr_fn2(void *arg);

void err_exit(int err, char *s);

int main(int argc, char const *argv[])
{
    int err;
    pthread_t tid1, tid2;
    void *tret;

    err = pthread_create(&tid1, NULL, thr_fn1, (void *)1);
    if (err != 0)
    {
        err_exit(err, "can't create thread 1");
    }

    err = pthread_create(&tid2, NULL, thr_fn2, (void *)1);
    if (err != 0)
    {
        err_exit(err, "can't create thread 2");
    }

    err = pthread_join(tid1, &tret);
    if (err != 0)
    {
        err_exit(err, "can't join with thread 1");
    }
    printf("thread 1 exit code %ld\n", (long)tret);

    err = pthread_join(tid2, &tret);
    if (err != 0)
    {
        err_exit(err, "can't join with thread 2");
    }
    printf("thread 2 exit code %ld\n", (long)tret);

    return 0;
}

// 线程清理函数
void cleanup(void *arg)
{
    printf("cleanup: %s\n", (char *)arg);
}

// 线程1 函数
void *thr_fn1(void *arg)
{
    sleep(2);
    printf("Thread 1 is running\n");
    pthread_cleanup_push(cleanup, "thread 1 first handler!");
    pthread_cleanup_push(cleanup, "thread 1 second handler!");
    printf("thread 1 push complete\n");
    if (arg)
    {
        pthread_exit((void *)1);
    }
    pthread_cleanup_pop(0);
    pthread_cleanup_pop(0);
    pthread_exit((void *)1);
}

// 线程2 函数
void *thr_fn2(void *arg)
{
    printf("Thread 2 is running\n");
    pthread_cleanup_push(cleanup, "thread 2 first handler!");
    pthread_cleanup_push(cleanup, "thread 2 second handler!");
    printf("thread 2 push complete\n");
    if (arg)
    {
        pthread_exit((void *)2);
    }
    pthread_cleanup_pop(0);
    pthread_cleanup_pop(0);
    pthread_exit((void *)2);
}

// 退出函数
void err_exit(int err, char *s)
{
    printf("%s\n", s);
    exit(err);
}