#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <fcntl.h>

#define BUF_SIZE 4096 // 每次读取字节数

int main(int argc, char const *argv[])
{
	int read_fd = open("./read.c", O_RDONLY);
	if (read_fd <0 ) {
		perror(argv[0]);
		return 0;
	}
	char buf[BUF_SIZE];
	int n;
	puts("读取的文件内容如下");
	while ((n=read(read_fd, buf, BUF_SIZE)) > 0)
	{
		puts(buf);
	}
	close(read_fd);
	return 0;
}
