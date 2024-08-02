#include <pwd.h>
#include <stddef.h>
#include <string.h>
#include <stdio.h>

struct passwd *mygetpwnam(const char *name)
{
    struct passwd *ptr;

    setpwent();
    while ((ptr = getpwent()) != NULL)
    {
        if (strcmp(name, ptr->pw_name) == 0)
        {
            break;
        }
    }
    endpwent();
    return (ptr);
}

int main(int argc, char const *argv[])
{
    struct passwd *pd = mygetpwnam("dxm");
    printf("uid = %d\n", pd->pw_uid);
    printf("gid = %d\n", pd->pw_gid);
    printf("home dir = %s\n", pd->pw_dir);
    struct passwd *pd1 = getpwnam("dxm");
    printf("uid = %d\n", pd1->pw_uid);
    printf("gid = %d\n", pd1->pw_gid);
    printf("home dir = %s\n", pd1->pw_dir);
    return 0;
}