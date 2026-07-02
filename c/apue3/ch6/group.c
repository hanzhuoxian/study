#include <grp.h>
#include <stddef.h>
#include <stdio.h>

int main(int argc, char const *argv[])
{
    setgrent();
    struct group *g;
    while ((g = getgrent()) != NULL)
    {
        printf("name %s gid %d mem %s\n", g->gr_name, g->gr_gid, g->gr_mem);
    }

    endgrent();
    return 0;
}
