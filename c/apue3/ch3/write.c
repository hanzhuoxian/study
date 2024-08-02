#include <unistd.h>
#include <stdio.h>
#include <string.h>
#include <fcntl.h>
#include <apue.h>
#include <stddef.h>

#define BUF_SIZE 4096 // 每次读取字节数
#define FILE_MODE (S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH)

int main(int argc, char const *argv[])
{
	int read_fd = open("./read.c", O_RDONLY);
	if (read_fd < 0)
	{
		perror(argv[0]);
		return 0;
	}
	int write_fd = open("./write.txt", O_CREAT | O_WRONLY , FILE_MODE);
	if (write_fd < 0)
	{
		perror(argv[0]);
		return 0;
	}
	char buf[BUF_SIZE];
	int n;
	while ((n = read(read_fd, buf, BUF_SIZE)) > 0)
	{
		if (write(write_fd, buf, n) != n)
		{
			perror(argv[0]);
		}
	}
	close(read_fd);
	close(write_fd);
	return 0;
}