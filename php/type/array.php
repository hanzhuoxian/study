<?php

if (true) {
    echo '';
}

$arr = [1,3,4,5];

array_splice($arr, 1, 0, 2);

var_dump($arr);
