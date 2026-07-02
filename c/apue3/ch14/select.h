#include <stdio.h>
#include <sys/select.h>

int main(int argc, char const *argv[])
{
    printf("%d", FD_SETSIZE);
    return 0;
}
