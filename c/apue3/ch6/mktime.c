#include <stdio.h>
#include <time.h>

int main(int argc, char const *argv[])
{
    struct tm *tmm;
    tmm->tm_year = 121;
    tmm->tm_mon = 11;
    tmm->tm_mday = 17;
    tmm->tm_hour = 01;
    tmm->tm_min = 01;
    tmm->tm_sec = 01;

    time_t t = mktime(tmm);
    if (t == -1)
    {
        perror("mktime error");
        return 1;
    }
    printf("time %ld\n", t);
    return 0;
}
