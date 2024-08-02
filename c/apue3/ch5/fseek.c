#include <stdio.h>

int main(int argc, char const *argv[])
{
    fseek(stdout, 0, SEEK_SET);
    return 0;
}
