#include <fcntl.h>
#include <apue.h>

#define RWRWRW (S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH)

int main(int argc, char const *argv[])
{
	if (creat("haha", RWRWRW) < 0)
	{
		err_sys("creat error for haha");
	}
	int oldumask = umask(0);
	printf("old umask : %d\n", oldumask);
	if (creat("foo", RWRWRW) < 0)
	{
		err_sys("creat error for foo");
	}
	oldumask = umask(S_IRGRP | S_IWGRP | S_IROTH | S_IWOTH);
	printf("old umask : %d\n", oldumask);
	if (creat("bar", RWRWRW) < 0)
	{
		err_sys("creat error for bar");
	}
	return 0;
}
