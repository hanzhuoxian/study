#include <unistd.h>
#include <stdio.h>
#include <fcntl.h>

int main(int argc, char const *argv[])
{
    if (symlink("./foo", "./foo.link") == -1)
    {
        perror(argv[0]);
        return 1;
    }
    if (unlink("./foo.link") < 0)
    {
        perror(argv[0]);
        return 2;
    }
    if (symlinkat("./foo", AT_FDCWD, "foo.link") == -1)
    {
        perror(argv[0]);
        return 1;
    }
    printf("end\n");
    return 0;
}
