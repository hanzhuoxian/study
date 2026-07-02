#include <stdio.h>
#include <time.h>
#include <stddef.h>

int main(int argc, char const *argv[])
{
    char buf[32];
    time_t t = time(NULL);
    struct tm *tmm = localtime(&t);
    char *t1 = strptime(buf, "%Y-%m-%d %H:%M:%S", tmm);
    if (t1 == NULL)
    {
        puts("error");
    }
    return 0;
}
