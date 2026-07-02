#include <unistd.h>
#include <fcntl.h>
#include <stdio.h>

int main(int argc, char *argv[])
{
	// 标准输出未关闭可以正常输出
	printf("start close stdout!\n");
	if (close(STDOUT_FILENO) < 0)
	{
		perror(argv[0]);
	}
	// 标准输出已经关闭，无法输出到终端
	printf("close stdout success!\n");
	return 0;
}
