CREATE TABLE anye_34 (
  `id` INT NOT NULL,
  `a` INT DEFAULT NULL,
  `b` INT DEFAULT NULL,
  KEY `a`(`a`)
) ENGINE = InnoDB;
DROP procedure idata;
DELIMITER;;
CREATE procedure idata() BEGIN declare i int;
set
  i = 1;
WHILE (i < 1000) do
INSERT INTO
  anye_34
VALUES(i, i, i);
SET
  i = i + 1;
END WHILE;
END;;
DELIMITER;
CALL idata();
CREATE TABLE anye_34_1 LIKE anye_34;
INSERT INTO
  anye_34_1 (
    SELECT
      *
    FROM
      anye_34
    WHERE
      id <= 100
  );
explain
select
  *
from
  anye_34_1 straight_join anye_34 on (anye_34_1.a = anye_34.b);
select
  *
from
  t1
  join t2 on(t1.a = t2.a)
  join t3 on (t2.b = t3.b)
where
  t1.c >= X
  and t2.c >= Y
  and t3.c >= Z;

  t1 c
  t2 a
  t3 b