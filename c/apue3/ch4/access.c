#include <unistd.h>
#include <fcntl.h>
#include <apue.h>

int main(int argc, char const *argv[])
{
	if (argc != 2)
	{
		printf("usage: access.app <pathname>");
		return 1;
	}

	if (access(argv[1], W_OK) < 0)
	{
		printf("access write error for %s\n", argv[1]);
	}
	else
	{
		printf("write access OK\n");
	}
	if (access(argv[1], R_OK) < 0)
	{
		printf("access error for %s\n", argv[1]);
	}
	else
	{
		printf("read access OK\n");
	}
	if (open(argv[1], O_RDONLY) < 0)
	{
		printf("open error for %s", argv[1]);
	}
	else
	{
		printf("open for reading OK\n");
	}
	return 0;
}
