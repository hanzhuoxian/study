#include <stdio.h>
#include <limits.h>
#include <float.h>

int main(int argc, char const *argv[])
{
    printf("head type limit\n");
    printf("signed char min %d ~ signed char max %d\n", SCHAR_MIN, SCHAR_MAX);
    printf("unsigned char min %d ~ unsigned char max %d\n", 0, UCHAR_MAX);
    printf("signed short min %d ~ signed short max %d\n", SHRT_MIN, SHRT_MAX);
    printf("unsigned short min %d ~ unsigned short max %d\n", 0, USHRT_MAX);
    printf("signed int min %d ~ signed int max %d\n", INT_MIN, INT_MAX);
    printf("unsigned int min %d ~ unsigned int max %d\n", 0, UINT_MAX);
    printf("signed long min %ld ~ signed long max %ld\n", LONG_MIN, LONG_MAX);
    printf("unsigned long min %d ~ unsigned long max %lu\n", 0, ULONG_MAX);
    printf("signed float min %f ~ signed float max %f\n", FLT_MIN, FLT_MAX);
    printf("float min %f ~ float max %f\n", FLT_MIN, FLT_MAX);
    printf("double min %f ~ double max %f\n", DBL_MIN, DBL_MAX);
    printf("long double min %Lf ~ long double max %Lf\n", LDBL_MIN, LDBL_MAX);

    printf("calc type limit\n");
    printf("signed char min %d ~ signed char max %d\n", (signed char)(1 << 7), (char)((unsigned char)~0 >> 1));
    printf("unsigned char min %d ~ unsigned char max %d\n", 0, (unsigned char)(1 << 7));
    printf("signed short min %d ~ signed short max %d\n", SHRT_MIN, SHRT_MAX);
    printf("unsigned short min %d ~ unsigned short max %d\n", 0, USHRT_MAX);
    printf("signed int min %d ~ signed int max %d\n", INT_MIN, INT_MAX);
    printf("unsigned int min %d ~ unsigned int max %d\n", 0, UINT_MAX);
    printf("signed long min %ld ~ signed long max %ld\n", LONG_MIN, LONG_MAX);
    printf("unsigned long min %d ~ unsigned long max %lu\n", 0, ULONG_MAX);
    printf("signed float min %f ~ signed float max %f\n", FLT_MIN, FLT_MAX);
    printf("float min %f ~ float max %f\n", FLT_MIN, FLT_MAX);
    printf("double min %f ~ double max %f\n", DBL_MIN, DBL_MAX);
    printf("long double min %Lf ~ long double max %Lf\n", LDBL_MIN, LDBL_MAX);
    return 0;
}
