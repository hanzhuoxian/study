#include <stdio.h>

int main(int argc, char const *argv[])
{
    FILE *fp = fopen("./fgetpos.c", "r");
    fpos_t spos = 10;
    fsetpos(fp, &spos);
    fpos_t pos;
    fgetpos(fp, &pos);
    printf("fgetpos %lld\n", pos);
    return 0;
}
