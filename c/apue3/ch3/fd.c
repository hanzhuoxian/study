#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>

int main(int argc, char *argv[])
{
	// 标准输入的文件描述符
	printf("Studio input file no: %d\n", STDIN_FILENO);
	// 标准输出的文件描述符
	printf("Studio output file no: %d\n", STDOUT_FILENO);
	// 标准出错的文件描述符
	printf("Studio error file no: %d\n", STDERR_FILENO);

	// 打开文件的文件描述符
	int fd = open("./fd.c", O_RDONLY);
	if (fd < 0)
	{
		printf("open fd.c failed!");
	}

	printf("open fd.c fd: %d\n", fd);
	return 0;
}