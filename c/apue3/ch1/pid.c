#include <apue.h>

int main(int argc, char *argv[])
{
    printf("hello world from process ID %d\n", getpid());
    return 0;
}