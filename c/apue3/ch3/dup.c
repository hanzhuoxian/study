#include <stdio.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
	int out_fd = dup(1);
	printf("stdout fd copy: %d\n",out_fd);
	int out_fd2 = dup2(1, 5);
	printf("stdout fd2 copy: %d\n",out_fd2);
	return 0;
}
