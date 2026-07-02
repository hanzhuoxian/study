# ·指定最低版本
cmake_minimum_required(VERSION 3.0.5)

# 设置项目名称
project(signal.app)

# 添加可执行文件
add_executable(signal.app sigprocmask.c)

project(signalusr.app)
add_executable(signalusr.app signalusr.c)

# kill 命令
project(kill.app)
add_executable(kill.app kill.c)

# alarm 命令
project(alarm.app)
add_executable(alarm.app alarm.c)

# sleep1 命令
project(sleep1.app)
add_executable(sleep1.app sleep1.c)

# sigpending 命令
project(sigpending.app)
add_executable(sigpending.app sigpending.c)

# mysignal 命令
project(mysignal.app)
add_executable(mysignal.app mysignal.c)

# siglongjmp 命令
project(siglongjmp.app)
add_executable(siglongjmp.app siglongjmp.c)

# sigsuspend
project(sigsuspend.app)
add_executable(sigsuspend.app sigsuspend.c)

# systemed
project(systemed.c)
add_executable(systemed.app systemed.c)

# psignal
project(psignal.c)
add_executable(psignal.app psignal.c)

# sleepmain
project(sleepmain.c)
add_executable(sleepmain.app sleepmain.c)
