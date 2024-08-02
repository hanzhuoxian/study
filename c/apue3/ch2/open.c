#include <fcntl.h>

int     open(const char *, int, ...) __DARWIN_ALIAS_C(open);

int     openat(int, const char *, int, ...) __DARWIN_NOCANCEL(openat) __OSX_AVAILABLE_STARTING(__MAC_10_10, __IPHONE_8_0);