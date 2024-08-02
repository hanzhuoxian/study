# ·指定最低版本
cmake_minimum_required(VERSION 3.0.5)

# print_pthread_id
project(print_pthread_id.c)
add_executable(printPthreadId.app print_pthread_id.c)

# exitcode
project(exitcode.c)
add_executable(exitcode.app exitcode.c)

# printfoo
project(printfoo.c)
add_executable(printfoo.app printfoo.c)

# cleanup
project(cleanup.c)
add_executable(cleanup.app cleanup.c)

# mutex
project(mutex.c)
add_executable(mutex.app mutex.c)