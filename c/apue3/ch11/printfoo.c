#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <stdlib.h>

// 定义结构体
struct foo
{
    int a, b, c, d;
};

// 打印foo的值
void printfoo(const char *s, const struct foo *fp)
{
    printf("%s", s);
    printf(" structure at 0x%lx\n", (unsigned long)fp);
    printf(" foo.a = %d\n", fp->a);
    printf(" foo.b = %d\n", fp->b);
    printf(" foo.c = %d\n", fp->c);
    printf(" foo.d = %d\n", fp->d);
}

// 线程1的入口函数
void *thr_fn1(void *arg)
{
    struct foo foo = {1, 2, 3, 4};
    printfoo("thread 1: ", &foo);
    pthread_exit((void *)&foo);

    // struct foo *foo = malloc(sizeof(foo));
    // foo->a = 1;
    // foo->b = 2;
    // foo->c = 3;
    // foo->d = 4;
    // printfoo("thread 1: ", foo);
    // pthread_exit((void *)foo);
}

// 线程2的入口函数
void *thr_fn2(void *arg)
{
    printf("thread 2: ID is %lu\n", (unsigned long)pthread_self());
    pthread_exit((void *)0);
}

// 退出函数
void err_exit(int err, char *s)
{
    printf("%s\n", s);
    exit(err);
}

int main(int argc, char const *argv[])
{
    int err;
    pthread_t tid1, tid2;
    struct foo *fp;

    err = pthread_create(&tid1, NULL, thr_fn1, NULL);
    if (err != 0)
    {
        err_exit(err, "can't create thread 1");
    }
    err = pthread_join(tid1, (void *)&fp);
    if (err != 0)
    {
        err_exit(err, "can't join with thread 1");
    }
    sleep(1);
    printf("parent starting second thread\n");
    err = pthread_create(&tid2, NULL, thr_fn2, NULL);
    if (err != 0)
    {
        err_exit(err, "can't create thread 2");
    }
    sleep(1);
    printfoo("parent: \n ", fp);

    return 0;
}
