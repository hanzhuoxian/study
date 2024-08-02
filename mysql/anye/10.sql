delimiter //
create procedure idata()
begin
  declare i int;
  set i=1;
  while(i<=100000)do
    insert into anye_10 values
    (i, i, i),
    (i+1, i+1, i+1),
    (i+2, i+2, i+2),
    (i+3, i+3, i+3),
    (i+4, i+4, i+4),
    (i+5, i+5, i+5),
    (i+6, i+6, i+6),
    (i+7, i+7, i+7),
    (i+8, i+8, i+8)
    ;
    set i=i+10;
  end while;
end//

SELECT count(*) FROM `anye_10`;

EXPLAIN select * from anye_10 where id between 10000 and 20000;