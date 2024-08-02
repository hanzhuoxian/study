#include <stdio.h>

// 定义整型常量
#define INT_CONST 123

// 定义长整型
#define LONG_CONST 123456L

// 定义浮点型常量
#define FLOAT_CONST 12.34f

// 定义浮点型常量
#define DOUBLE_CONST 12.34

// 水平制表符
#define TAB '\011'

int main(int argc, char const *argv[])
{
    printf("int const %d\n", INT_CONST);
    printf("long const %ld\n", LONG_CONST);
    printf("float const %f\n", FLOAT_CONST);
    printf("double const %f\n", DOUBLE_CONST);
    printf("%c\n", TAB);
    return 0;
}
