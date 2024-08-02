#!/usr/bin/env bash

# 一个标准的 C 语言语法的 `for` 循环
for ((i=0;i<10;i++))
do
    echo $i
done

# 标准的 `for in` 循环
for i in 1 2 3
do
    echo $i
done

# 列表由通配符产生的 for 循环
for conf in /etc/*conf;do
    echo "$conf"
done

# for in 遍历数组
declare -a arr=("element1" "element2" "element3")

for element in "${arr[@]}";do
    echo "$element"
done

# for in 省略 in list
forignoreinlist() {
    # write in list
    for i in "$@";do
        echo "$i"
    done
    # ignore in list
    for i;do
        echo "$i"
    done
}

forignoreinlist a b c d