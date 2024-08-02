# mysql 实战

## 本项目的文章地址

[文章地址](https://www.yuque.com/anyezhixue/tvk0lc)  
[https://www.yuque.com/anyezhixue/tvk0lc](https://www.yuque.com/anyezhixue/tvk0lc)

## 准备环境

- docker
- docker-compose

## 使用docker创建mysql的实验环境

```sh
# 进入目录
cd docker/mysql

# 启动mysql服务
make brun
```

## 进入mysql服务器

```sh
docker exec -it mysql_mysql_service_1 /bin/bash
```

## 查看慢查日志

- 由于long_query_time = 0的设置，导致每条sql都会记录,本项目的宗旨就是研究mysql，所以要观察每条sql的执行情况

```sh
tail -f /var/log/docker_log/mysql/slow.log
```

## 查看binlog日志

```sh
mysqlbinlog /var/log/mysql-binlog/mysql-binlog.*
```

## 本项目主要学习自极客时间的《MySQL实战45讲》
