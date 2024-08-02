# ·指定最低版本
cmake_minimum_required(VERSION 3.0.5)

# noblock_io
project(noblock_io)
add_executable(noblock_io.app noblock_io.c)

# readlock
project(readlock)
add_executable(readlock.app readlock.c)

# deadlock
project(deadlock)
add_executable(deadlock.app deadlock.c)

# lock
project(lock)
add_executable(lock.app lock.c)

# forcelock
project(forcelock)
add_executable(forcelock.app forcelock.c)