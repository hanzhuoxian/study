CREATE TABLE anye_8
(
    id int NOT NULL,
    k int default null,
    primary key (id)
)ENGINE=InnoDB;

INSERT INTO anye_8(id,k) values(1,1),(2,2);


CREATE TABLE `anye_8_1` (
  `id` int(11) NOT NULL,
  `c` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;
insert into anye_8_1(id, c) values(1,1),(2,2),(3,3),(4,4);

-- c 清不空构造场景
update anye_8_1 set c=0 where id=c; 

-- 执行上面的语句前先执行这个
update anye_8_1 set c=c+1 where 1=1; 