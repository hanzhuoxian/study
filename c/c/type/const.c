#include <stdio.h>

void changeArray(int const arr[4])
{
    arr[0] = 1;
}

int main(int argc, char const *argv[])
{
    int i = 1;
    i = 2;
    int const j = 3;
    // j = 4; // 不能赋值
    int const week[4] = {1, 2, 3, 4};
    // week[0] = 1; //不能赋值
    for (i = 0; i < 4; i++)
    {
        printf("%d\n", week[i]);
    }
    changeArray(week);
    return 0;
}
