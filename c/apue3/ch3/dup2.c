#include <stdio.h>
#include <unistd.h>

// 实现与dup2功能相同的函数
int mydup2(int fd, int fd2);

int main(int argc, char const *argv[])
{
	int fd5 = mydup2(1, 5);
	printf("fd5 : %d", fd5);
	return 0;
}

int mydup2(int fd, int fd2)
{
	if (fd == fd2) {
		return fd2;
	}

	close(fd2);
	access()

}
