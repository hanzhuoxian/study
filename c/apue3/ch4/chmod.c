#include <stdio.h>
#include <sys/stat.h>
#include <strings.h>

int main(int argc, char const *argv[])
{
	struct stat statbuf;
	if (stat("foo", &statbuf) < 0)
	{
		perror("stat error for foo");
	}
	if (chmod("foo", (statbuf.st_mode & ~S_IXGRP) | S_ISGID) < 0)
	{
		perror("stat error for foo");
	}

	if (chmod("bar", S_IRUSR | S_IWUSR | S_IRGRP | S_IWGRP) < 0)
	{
		perror("stat error for bar");
	}
	return 0;
}
