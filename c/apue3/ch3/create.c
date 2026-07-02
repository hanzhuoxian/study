#include <string.h>
#include <stdio.h>
#include <fcntl.h>

int main(int argc, char *argv[])
{
	int fd = creat("./create.c", O_WRONLY);
	if (fd < 0)
	{
		perror(argv[0]);
	}
	close(fd);
	return 0;
}