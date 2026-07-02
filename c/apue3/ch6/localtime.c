#include <time.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    time_t t = time(NULL);
    struct tm *tmm = localtime(&t);
    if (tmm == NULL)
    {
        perror("localtime error");
    }
    printf("%d-%d-%d %d:%d:%d\n", 1900 + tmm->tm_year, tmm->tm_mon, tmm->tm_mday, tmm->tm_hour, tmm->tm_min, tmm->tm_sec);
    return 0;
}
