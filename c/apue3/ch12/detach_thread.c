#include <pthread.h>
#include <stdio.h>
#include <unistd.h>
#include <limits.h>

void print_err(char *str, int err)
{
    if (err != 0)
    {
        printf("%s, errno is %d\n", str, err);
    }
}

int makethread(void *(*fn)(void *), void *arg)
{
    int err;
    pthread_t tid;
    pthread_attr_t attr;
    size_t stacksize;
    void *stackaddr;
    int detachstate;
    size_t guardsize;
    size_t setstacksize;

    err = pthread_attr_init(&attr);
    if (err != 0)
    {
        return (err);
    }
    err = pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
    if (err == 0)
    {
        err = pthread_create(&tid, &attr, fn, arg);
    }

    printf("PTHREAD_STACK_MIN is %ld\n", sysconf(PTHREAD_STACK_MIN));
    setstacksize = 524288 * 2;
    err = pthread_attr_setstacksize(&attr, setstacksize);
    print_err("pthread_attr_setstacksize is failed", err);
    err = pthread_attr_getstack(&attr, &stackaddr, &stacksize);
    print_err("pthread_attr_getstack is failed", err);
    printf("thread stack size is %zu\n", stacksize);
    err = pthread_attr_getstacksize(&attr, &stacksize);
    print_err("pthread_attr_getstacksize is failed", err);
    printf("thread stack size is %zu\n", stacksize);
    err = pthread_attr_getdetachstate(&attr, &detachstate);
    print_err("pthread_attr_getdetachstate is failed", err);
    printf("thread detach state is :%d\n", detachstate);
    err = pthread_attr_setguardsize(&attr, setstacksize);
    print_err("pthread_attr_setguardsize is failed", err);
    err = pthread_attr_getguardsize(&attr, &guardsize);
    print_err("pthread_attr_getguardsize is failed", err);
    printf("thread guardsize is : %zu\n", guardsize);
    pthread_attr_destroy(&attr);
    return (err);
}
void *thr_fn(void *arg)
{
    printf("thread 1 is started\n");
    return ((void *)0);
}

int main(int argc, char const *argv[])
{
    makethread(thr_fn, NULL);
    sleep(2);
    return 0;
}
