#include <stdio.h>
#include <pthread.h>
#include "getenv_r.h"
#include "getenv.h"
#include "getenv_sepc.h"

const int LEN = sizeof(char) * 1000;

static void *thr_fn(void *arg)
{
    printf("start get env %s\n", (char *)arg);
    char *env_name = (char *)arg;
    char buf[LEN];
    char *env_value_p = buf;
    // 线程不安全
    printf("unsafe thread %s is %s\n", (char *)arg, getenv((char *)arg));

    // 线程安全
    int err = getenv_r(env_name, env_value_p, LEN);
    if (err != 0)
    {
        printf("can't get env for %s", (char *)arg);
        return ((void *)1);
    }
    printf("safe thread %s:%s\n", (char *)arg, env_value_p);

    // 特定数据线程安全
    printf("spec thread %s:%s\n", (char *)arg, getenv_sepc((char *)arg));
    return ((void *)0);
}

int main(int argc, char const *argv[])
{
    pthread_t tid1, tid2;
    int err;
    void *ret;
    err = pthread_create(&tid1, NULL, thr_fn, "LANG");
    if (err != 0)
    {
        printf("can't create thread 1\n");
    }

    err = pthread_create(&tid2, NULL, thr_fn, "HOME");
    if (err != 0)
    {
        printf("can't create thread 2\n");
    }
    err = pthread_join(tid1, &ret);
    if (err != 0)
    {
        printf("can't join thread 1");
    }
    err = pthread_join(tid2, &ret);
    if (err != 0)
    {
        printf("can't join thread 2");
    }
    return 0;
}
