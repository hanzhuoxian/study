#include <pthread.h>
#include <stdlib.h>
#include <stdio.h>

struct foo
{
    int f_count;
    int f_id;
    pthread_mutex_t f_lock;
};

struct foo *foo_alloc(int id)
{
    struct foo *fp;
    if ((fp = malloc(sizeof(struct foo))) != NULL)
    {
        fp->f_count = 1;
        fp->f_id = id;
        if (pthread_mutex_init(&fp->f_lock, NULL) != 0)
        {
            free(fp);
            return NULL;
        }
    }

    return fp;
}

void foo_hold(struct foo *fp)
{
    pthread_mutex_lock(&fp->f_lock);
    fp->f_count++;
    pthread_mutex_unlock(&fp->f_lock);
}

void foo_rele(struct foo *fp)
{
    pthread_mutex_lock(&fp->f_lock);
    if (--fp->f_count == 0)
    {
        pthread_mutex_unlock(&fp->f_lock);
        pthread_mutex_destroy(&fp->f_lock);
        free(fp);
    }
    else
    {
        pthread_mutex_unlock(&fp->f_lock);
    }
}

void printfoo(struct foo *fp)
{
    printf("thread %lu : id=%d,count=%d\n", (unsigned long)pthread_self(), fp->f_id, fp->f_count);
}

void *thr_fn1(void *arg)
{
    struct foo *myfoo = (struct foo *)(arg);
    foo_hold(myfoo);
    printfoo(myfoo);
    pthread_mutex_lock(&myfoo->f_lock);
    myfoo->f_id = 2;
    printfoo(myfoo);
    pthread_mutex_unlock(&myfoo->f_lock);
    foo_rele(myfoo);
    return ((void *)1);
}

void *thr_fn2(void *arg)
{
    struct foo *myfoo = (struct foo *)(arg);
    foo_hold(myfoo);
    printfoo(myfoo);
    pthread_mutex_lock(&myfoo->f_lock);
    myfoo->f_id = 3;
    printfoo(myfoo);
    pthread_mutex_unlock(&myfoo->f_lock);
    foo_rele(myfoo);
    return ((void *)2);
}

void err_exit(int err, char *s)
{
    printf("%s\n", s);
    exit(err);
}

int main(int argc, char const *argv[])
{
    int err;
    pthread_t tid1, tid2;
    void *tret;
    struct foo *myfoo = foo_alloc(1);

    err = pthread_create(&tid1, NULL, thr_fn1, myfoo);
    if (err != 0)
    {
        err_exit(err, "can't create thread 1");
    }

    err = pthread_create(&tid2, NULL, thr_fn2, myfoo);
    if (err != 0)
    {
        err_exit(err, "can't create thread 1");
    }

    err = pthread_join(tid1, &tret);
    if (err != 0)
    {
        err_exit(err, "can't join with thread 1");
    }
    printf("thread 1 is finished!\n");

    err = pthread_join(tid2, &tret);
    if (err != 0)
    {
        err_exit(err, "can't join with thread 1");
    }
    printf("thread 2 is finished!\n");
    return 0;
}
