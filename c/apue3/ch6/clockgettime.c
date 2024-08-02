#if defined(I_OS_LINUX)

#endif
#include <time.h>
#include <stdio.h>
int main(int argc, char const *argv[])
{
    struct timespec tsp;
    int t = clock_gettime(CLOCK_REALTIME, &tsp);
    if (t != 0)
    {
        perror("clock gettime error");
        return 1;
    }
    printf("timetsp.tv_sec: %ld\n", tsp.tv_sec);
    printf("timetsp.tv_sectv_nsec: %ld\n", tsp.tv_nsec);
    return 0;
}
