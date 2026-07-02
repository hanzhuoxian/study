#include <stdio.h>

int main(int argc, char const *argv[])
{
    // 定位流
    FILE *fp = fopen("./Makefile", "r");
    int seekPos = fseek(fp, 3L, SEEK_SET);
    if (seekPos == -1)
    {
        printf("fseek error\n");
    }
    long tell = ftell(fp);
    printf("tell %ld\n", tell);
    return 0;
}
