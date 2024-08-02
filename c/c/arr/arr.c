#include <stdio.h>

void change(int *arr)
{
    arr[0] = 2;
    printf("arr address: %p\n", arr);
    arr++;
    printf("arr++ address: %p\n", arr);
}

int main(int argc, char const *argv[])
{
    int arr[5] = {1, 2, 3, 4, 5};
    printf("arr address: %p\n", arr);
    change(arr);
    for (int i = 0; i < 5; i++)
    {
        printf("%d,", arr[i]);
    }
    printf("\n");
    return 0;
}
