#include <limits.h>
#include <pthread.h>
#include <stdio.h>
#include <string.h>

#define MAXSTRINGSZ 4096

static pthread_key_t key;
static pthread_once_t init_done_spec = PTHREAD_ONCE_INIT;
pthread_mutex_t env_mutex_spec = PTHREAD_MUTEX_INITIALIZER;

extern char **environ;

static void thread_init_spec(void)
{
    pthread_key_create(&key, free);
}

char *getenv_sepc(char *name)
{
    int i, len;
    char *envbuf;

    // 初始化线程数据，不论几个线程只初始化一次
    pthread_once(&init_done_spec, thread_init_spec);

    pthread_mutex_lock(&env_mutex_spec);

    envbuf = (char *)pthread_getspecific(key);

    if (envbuf == NULL) // 没有特定数据分配并绑定数据
    {
        envbuf = (char *)malloc(MAXSTRINGSZ);
        if (envbuf == NULL) // 分配内存失败
        {
            pthread_mutex_unlock(&env_mutex_spec);
            return (NULL);
        }
        pthread_setspecific(key, &envbuf);
    }

    len = strlen(name);
    for (i = 0; environ[i] != NULL; i++)
    {
        if ((strncmp(name, environ[i], len)) == 0 &&
            environ[i][len] == '=')
        {
            strncpy(envbuf, &environ[i][len + 1], MAXSTRINGSZ - 1);
            pthread_mutex_unlock(&env_mutex_spec);
            return envbuf;
        }
    }
    pthread_mutex_unlock(&env_mutex_spec);
    return (NULL);
}