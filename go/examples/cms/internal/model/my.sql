CREATE TABLE t1
(
    id int primary key auto_increment,
    name varchar(255),
    key idx_name(name)
);
alter table t1 add age int;

begin;
update t1 set age = age + 1 where id = 1;

begin;
update t1 set age = age + 1 where id = 2;

update t1 set age = age + 2 where id = 1;

insert into t1(name) values('zhangsan'),('lisi');