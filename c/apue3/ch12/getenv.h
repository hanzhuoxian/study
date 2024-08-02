#include <stdio.h>
#include <string.h>

#define MAXSTRINGGSZ 4096

static char envbuf[MAXSTRINGGSZ];

extern char **environ;

char *getenv(const char *name)
{
    int i, len;
    len = strlen(name);
    for (i = 0; environ[i] != NULL; i++)
    {
        if ((strncmp(name, environ[i], len) == 0) && (environ[i][len] == '='))
        {
            strncpy(envbuf, &environ[i][len + 1], MAXSTRINGGSZ - 1);
            return (envbuf);
        }
    }
    return (NULL);
}