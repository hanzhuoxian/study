#include <fcntl.h>
#include <stdio.h>
#include <unistd.h>
#include <string.h>
#include <stdlib.h>

int main(int argc, char const *argv[])
{
	int stdin_copy_fd = fcntl(STDIN_FILENO, F_DUPFD);
	printf("stdin copy fd : %d\n", stdin_copy_fd);

	int own = fcntl(STDIN_FILENO, F_GETOWN);
	printf("own : %d\n", own);

	int val;
	if (argc != 2)
	{
		printf("usage: fcntl.app <descriptor#>\n");
		return 0;
	}

	if ((val = fcntl(atoi(argv[1]), F_GETFL, 0)) < 0)
	{
		printf("fcntl error for fd %d", atoi(argv[1]));
	}

	switch (val & O_ACCMODE)
	{
	case O_RDONLY:
		printf("read only");
		break;
	case O_WRONLY:
		printf("write only");
		break;
	case O_RDWR:
		printf("read write");
		break;

	default:
		printf("unknown access mode");
		break;
	}

	if (val & O_APPEND) {
		printf(", append");
	}
	if (val & O_NONBLOCK) {
		printf(", nonblocking");
	}
	if (val & O_SYNC) {
		printf(", synchronous writes");
	}
#if !defined(_POSIX_C_SOURCE) && defined(O_FSYNC) && (O_FSYNC != O_SYNC)
	if (val & O_FSYNC) {
		printf(", synchronous writes");
	}
#endif

	putchar('\n');
	return 0;
}
