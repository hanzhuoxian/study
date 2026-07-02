#include <stdio.h>

#define BUFSIZE 1024

FILE *open_data(void)
{
    FILE *fp;
    char databuf[BUFSIZE]; // setvbuf makes this the stdio buffer

    if ((fp = fopen("Makefile", "r")) == NULL)
        return NULL;

    if ((setvbuf(fp, databuf, _IOLBF, BUFSIZE) != 0))
        return NULL;
    return fp;
}
int main(int argc, char const *argv[])
{

    return 0;
}
