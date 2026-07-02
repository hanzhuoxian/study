#include <unistd.h>
#include <stdio.h>

// lseek 仅当文件偏移量记录在内核中，不会引起I/O操作
int main(int argc, char const *argv[])
{
	// 查看当前文件偏移量
	off_t cur_seek = lseek(STDIN_FILENO, 0, SEEK_CUR);
	printf("file seek : %lld\n", cur_seek);
	// 设置标准输入文件偏移量
	off_t start_seek = lseek(STDIN_FILENO, 0, SEEK_SET);
	if (start_seek == -1)
	{
		printf("can't seek stdin");
	}
	printf("file seek : %lld\n", start_seek);
	return 0;
}
