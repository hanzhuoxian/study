#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <unistd.h>

pthread_mutex_t lock1 = PTHREAD_MUTEX_INITIALIZER;
pthread_mutex_t lock2 = PTHREAD_MUTEX_INITIALIZER;

/**
 * atfork 的prepare函数
 **/
void prepare(void);
/**
 * atfork 的parent函数
 **/
void parent(void);
/**
 * atfork 的child函数
 **/
void child(void);
/**
 * 线程
 **/
void *thr_fn(void *arg);

int main(int argc, char const *argv[])
{
    int err;
    pid_t pid;
    pthread_t tid;
    if ((err = pthread_atfork(prepare, parent, child)) != 0)
    {
        printf("can't install fork handlers, error is %d\n", err);
    }
    if ((err = pthread_create(&tid, NULL, thr_fn, (void *)0)) != 0)
    {
        printf("can't create thread , error is %d\n", err);
    }
    sleep(2);
    printf("parent about to fork...\n");
    if ((pid = fork()) < 0)
    {
        printf("can't fork failed , error is %d\n", err);
    }
    else if (pid == 0)
    {
        printf("child returned form fork\n");
    }
    else
    {
        printf("parent returned form fork\n");
    }
    return 0;
}

void prepare(void)
{
    int err;
    printf("prepareing locks...\n");
    if ((err = pthread_mutex_lock(&lock1)) != 0)
    {
        printf("can't lock lock1 in parent handler , error is %d\n", err);
    }
    if ((err = pthread_mutex_lock(&lock2)) != 0)
    {
        printf("can't lock lock2 in parent handler , error is %d\n", err);
    }
}
void parent(void)
{
    int err;
    printf("parent unlocking locks...\n");
    if ((err = pthread_mutex_unlock(&lock1)) != 0)
    {
        printf("can't unlock lock1 in parent handler , error is %d\n", err);
    }
    if ((err = pthread_mutex_unlock(&lock2)) != 0)
    {
        printf("can't unlock lock2 in parent handler , error is %d\n", err);
    }
}
void child(void)
{
    int err;
    printf("child unlocking locks...\n");
    if ((err = pthread_mutex_unlock(&lock1)) != 0)
    {
        printf("can't unlock lock1 in child handler , error is %d\n", err);
    }
    if ((err = pthread_mutex_unlock(&lock2)) != 0)
    {
        printf("can't unlock lock2 in child handler , error is %d\n", err);
    }
}
void *thr_fn(void *arg)
{
    printf("thread started ...\n");
    pause();
}