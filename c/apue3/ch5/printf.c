#include <stdio.h>
#include <stdarg.h>
#include <stdlib.h>

char *newfmt(const char *fmt, ...)
{
    char *p;
    va_list ap;
    if ((p = malloc(128)) == NULL)
        return (NULL);
    va_start(ap, fmt);
    (void)vsnprintf(p, 128, fmt, ap);
    va_end(ap);
    return (p);
}

int main(int argc, char const *argv[])
{
    printf("%s\n", newfmt("number is %d %d", 1, 2));
    return 0;
}
