#include <time.h>
#include <stdio.h>

void printNowTime()
{
    time_t *cur_time;
    struct tm *p;

    time(cur_time);
    gmtime(cur_time);

    char time_str[64];
    strftime(time_str, sizeof(time_str), "%Y-%m-%d %H:%M:%S", p);
    printf(time_str);
}