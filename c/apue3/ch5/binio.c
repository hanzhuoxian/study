#include <stdio.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    float data[10];
    for (int i = 0; i < 10; i++)
    {
        data[i] = i * 20;
    }
    FILE *fp = fopen("./binio.bin", "w+");
    if (fwrite(&data, sizeof(float), 10, fp) != 10)
    {
        printf("write error\n");
    }
    // 将文件指针移至文件开头
    printf("file pos %ld\n", ftell(fp));
    // fseek(fp, 0, SEEK_SET);
    rewind(fp);
    printf("file pos %ld\n", ftell(fp));

    float data1[10];
    int rsize = fread(&data1, sizeof(float), 10, fp);
    if (rsize != 10)
    {
        printf("read failed, size %d\n", rsize);
    }
    for (int j = 0; j < 10; j++)
    {
        printf("%f\n", data1[j]);
    }
    return 0;
}
