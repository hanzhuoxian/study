#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>

pthread_mutexattr_t attr;
pthread_mutex_t mutex;
int sum = 0;

void *thr_fn(void *);
void *thr_fn1(void *);

int main(int argc, char const *argv[])
{
    pthread_mutexattr_init(&attr);
    int mutex_type;
    pthread_mutexattr_gettype(&attr, &mutex_type);
    printf("pthread_mutexattr_gettype %d\n", mutex_type);
    pthread_mutexattr_settype(&attr, PTHREAD_MUTEX_RECURSIVE);
    pthread_mutexattr_gettype(&attr, &mutex_type);
    printf("pthread_mutexattr_gettype %d\n", mutex_type);
    pthread_mutex_init(&mutex, &attr);
    pthread_t tid;
    int err;
    err = pthread_create(&tid, NULL, &thr_fn, "thread 1");
    if (err != 0)
    {
        printf("pthread_create is failed\n");
        exit(2);
    }
    pthread_join(tid, NULL);
    err = pthread_create(&tid, NULL, &thr_fn1, "thread 2");
    if (err != 0)
    {
        printf("pthread_create is failed\n");
        exit(3);
    }
    return 0;
}

void *thr_fn(void *arg)
{
    pthread_mutex_lock(&mutex);
    sum++;
    pthread_mutex_unlock(&mutex);
    printf("%s sum is %d\n", (char *)arg, sum);
    return ((void *)0);
}

void *thr_fn1(void *arg)
{
    pthread_mutex_lock(&mutex);
    sum++;
    pthread_mutex_unlock(&mutex);
    printf("%s sum is %d\n", (char *)arg, sum);
    return ((void *)0);
}