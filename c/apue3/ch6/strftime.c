#include <stdio.h>
#include <time.h>

int main(int argc, char const *argv[])
{
    time_t t = time(NULL);
    char buf[32];
    struct tm *tmm = localtime(&t);
    strftime(buf, 32, "%Y-%m-%d %H:%M:%S", tmm);
    printf("%s\n", buf);
    return 0;
}
