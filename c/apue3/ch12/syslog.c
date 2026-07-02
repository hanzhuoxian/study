#include <stdio.h>
#include <syslog.h>

int main(int argc, char const *argv[])
{
    syslog(LOG_ERR, "I am is syslog error");
    return 0;
}
