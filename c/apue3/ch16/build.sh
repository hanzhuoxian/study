if [ $# -eq 1 ];then
cmake -B build && cd ./build && make && "./$1.app"
else
cmake -B build && cd ./build && make
fi