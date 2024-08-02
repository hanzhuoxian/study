DROP TABLE anye.anye_user;
CREATE TABLE anye_user (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  email varchar(64),
  INDEX idx_email(email)
) engine = InnoDB;

INSERT INTO
  anye_user (email)
VALUES
  ("zhangsh1234@163.com"),
  ("zhangssxyz@163.com"),
  ("zhangsy1998@163.com"),
  ("zhangszhzsz@163.com");

EXPLAIN  SELECT email FROM anye.anye_user WHERE email='zhangsh1234@163.com';


DROP TABLE IF EXISTS  anye.anye_user1;
CREATE TABLE anye_user1 (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  email varchar(64),
  INDEX idx_email(email(6))
) engine = InnoDB;

INSERT INTO
  anye_user1 (email)
VALUES
  ("zhangsh1234@163.com"),
  ("zhangssxyz@163.com"),
  ("zhangsy1998@163.com"),
  ("zhangszhzsz@163.com");

ALTER TABLE anye.anye_user1 DROP  INDEX  idx_email;
ALTER  TABLE anye.anye_user1 ADD  INDEX idx_email(email(7));

EXPLAIN SELECT * 
FROM anye.anye_user1 
FORCE INDEX (idx_email)
WHERE email='zhangsh1234@163.com';


select 
count(distinct email) as L,
count(distinct left(email,4)) as L4,
count(distinct left(email,5)) as L5,
count(distinct left(email,6)) as L6,
count(distinct left(email,7)) as L7,
count(distinct left(email,8)) as L8
from anye_user;