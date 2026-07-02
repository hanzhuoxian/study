#include <stdio.h>
#include <time.h>

int main(int argc, char const *argv[])
{
    time_t tt = time(NULL);
    printf("%ld\n", (long)tt);
    return 0;
}
