# ·指定最低版本
cmake_minimum_required(VERSION 3.0.5)

# daemonize
project(daemonize)
add_executable(daemonize.app daemonize.c)

# pthread_limit
project(pthreadLimit)
add_executable(pthreadLimit.app pthread_limit.c)
