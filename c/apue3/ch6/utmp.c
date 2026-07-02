#include <utmp.h>
#include <stdio.h>
#include <stddef.h>

int main(int argc, char const *argv[])
{
    struct utmp *ut;
    setutent();
    while ((ut = getutent()) != NULL)
    {
        printf("utmp.ut_host %s\n", ut->ut_host);
        printf("utmp.ut_id %s\n", ut->ut_id);
        printf("utmp.ut_user %s\n", ut->ut_user);
    }

    endutent();
    return 0;
}
