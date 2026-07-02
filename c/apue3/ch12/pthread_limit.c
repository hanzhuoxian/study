#include <unistd.h>
#include <stdio.h>
#include <pthread.h>

int main(int argc, char const *argv[])
{
    printf("_SC_THREAD_DESTRUCTOR_ITERATIONS = %ld\n", sysconf(_SC_THREAD_DESTRUCTOR_ITERATIONS));
    printf("_SC_THREAD_KEYS_MAX = %ld\n", sysconf(_SC_THREAD_KEYS_MAX));
    printf("_SC_THREAD_STACK_MIN = %ld\n", sysconf(_SC_THREAD_STACK_MIN));
    printf("_SC_THREAD_THREADS_MAX = %ld\n", sysconf(_SC_THREAD_THREADS_MAX));
    return 0;
}