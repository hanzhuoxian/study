#include <shadow.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    setspent();
    struct spwd *sp = getspnam("madison");
    if (sp == NULL)
    {
        perror("getspnam error");
        return 1;
    }
    printf("spwd.sp_namp: %s\n", sp->sp_namp);
    printf("spwd.sp_pwdp: %s\n", sp->sp_pwdp);
    printf("spwd.sp_lstchg: %ld\n", sp->sp_lstchg);
    printf("spwd.sp_min: %ld\n", sp->sp_min);
    printf("spwd.sp_max: %ld\n", sp->sp_max);
    printf("spwd.sp_warn: %ld\n", sp->sp_warn);
    printf("spwd.sp_inact: %ld\n", sp->sp_inact);
    printf("spwd.sp_expire: %ld\n", sp->sp_expire);
    printf("spwd.sp_flag: %ld\n", sp->sp_flag);
    endspent();
    return 0;
}