<?php

function gen_one_to_three()
{
    for ($i = 1; $i <= 3; $i++) {
        //注意变量$i的值在不同的yield之间是保持传递的。
        yield [$i];
    }
    echo 'ddd' . PHP_EOL;
}

$generator = gen_one_to_three();

foreach ($generator as $value) {
    var_dump($value);
}
