CREATE TABLE `anye_30` (
  `id` int(11) NOT NULL,
  `c` int(11) DEFAULT NULL,
  `d` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `c` (`c`)
) ENGINE = InnoDB;
INSERT INTO
  anye_30
VALUES
  (0, 0, 0),(5, 5, 5),(10, 10, 10),(15, 15, 15),(20, 20, 20),(25, 25, 25);
-- 悲观锁的查询语句
SELECT
  *
FROM
  anye_30
WHERE
  id > 9
  AND id < 12
ORDER BY
  id DESC FOR
UPDATE;
-- 加锁范围 主键索引 (10,15),(5,10],(0,5]
  -- 加锁范围 c索引无锁
UPDATE
  anye_30
SET
  d = d + 1
WHERE
  id = 0;
INSERT INTO
  anye.anye_30
VALUES(3, 3, 3);
UPDATE
  anye_30
SET
  d = d + 1
WHERE
  c = 5;
--
select
  id
from
  anye_30
where
  c in(5, 20, 10) ORDER BY c asc lock in share mode;
-- 加锁范围 (0,5],(5,10), (5,10],(10,15), (15,20],(20,25)
select
  id
from
  anye_30
where
  c in(5, 20, 10)
order by
  c desc for
update;
-- 加锁范围  (15,20],(20,25), (5,10],(10,15), (0,5],(5,10)

CREATE TABLE `anye_30_1` (
  `id` int(11) NOT NULL,
  `c` int(11) DEFAULT NULL,
  `d` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `c` (`c`)
) ENGINE = InnoDB;


SELECT * FROM anye_30_1 WHERE id = 10 FOR UPDATE;
INSERT INTO anye_30_1 VALUES(5,5,5);
INSERT INTO anye_30_1 VALUES(15,15,15);
INSERT INTO anye_30_1 VALUES(10,10,10);
INSERT INTO anye_30_1 VALUES(-15,-15,-15);