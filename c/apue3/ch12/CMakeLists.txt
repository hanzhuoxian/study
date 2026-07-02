# ·指定最低版本
cmake_minimum_required(VERSION 3.0.5)

# pthread_limit
project(pthreadLimit)
add_executable(pthreadLimit.app pthread_limit.c)

# detach_thread
project(detachThread)
add_executable(detachThread.app detach_thread.c)

# mutexattr
project(mutexattr)
add_executable(mutexattr.app mutexattr.c)

# getenvmain
project(getenvmain)
add_executable(getenvmain.app getenvmain.c)

# sigmask
project(sigmask)
add_executable(sigmask.app sigmask.c)

# wait
project(wait)
add_executable(wait.app wait.c)

# atfork
project(atfork)
add_executable(atfork.app atfork.c)

# syslog
project(syslog)
add_executable(syslog.app syslog.c)