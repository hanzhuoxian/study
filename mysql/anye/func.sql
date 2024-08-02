
DROP PROCEDURE cityes;
delimiter //

CREATE PROCEDURE cityes(IN country CHAR(3), OUT cities CHAR(5))
BEGIN
    SET @cities = concat("[", country , "]");
    SELECT @cities;
END//

delimiter ;

CREATE FUNCTION hello(s CHAR(20))
RETURNS CHAR(50) DETERMINISTIC
RETURN CONCAT('HELLO,', s, '!');