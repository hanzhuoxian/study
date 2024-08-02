#include <stdio.h>
#include <wchar.h>
#include <errno.h>
#include <fcntl.h>
#include <strings.h>

int main(int argc, char const *argv[])
{
    FILE *fp = fopen("./dddd", "r");
    printf("errno: %d\n", errno);
    fwide(fp, 1);
    printf("errno: %d\n", errno);
    perror("errno");

    return 0;
}
