-- docker exec -it mysql_mysql_service_1 /bin/bash
-- (5,10),(10,15],(15,20],(20,25)
CREATE TABLE `anye_21` (
  `id` int(11) NOT NULL,
  `c` int(11) DEFAULT NULL,
  `d` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `c` (`c`)
) ENGINE = InnoDB;
insert into
  anye_21
values(0, 0, 0),(5, 5, 5),(10, 10, 10),(15, 15, 15),(20, 20, 20),(25, 25, 25);
SELECT
  *
FROM
  anye.anye_21
WHERE
  id = 0;
SELECT
  *
FROM
  anye_21
WHERE
  c >= 15
  AND c <= 20
ORDER BY
  c DESC LOCK IN SHARE MODE;
UPDATE
  anye_21
SET
  d = d + 1
WHERE
  c = 5;
INSERT INTO
  anye_21
values(7, 7, 7);
INSERT INTO
  anye_21
values(11, 11, 11);
UPDATE
  anye_21
SET
  d = d + 1
WHERE
  c = 11;
INSERT INTO
  anye_21
values(24, 24, 24);
UPDATE
  anye_21
SET
  d = d + 1
WHERE
  c = 25;

SHOW variables LIKE '%innodb_read_only%';

