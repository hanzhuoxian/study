<?php

function getData() {
    $data = [1,2,3,4,5,6,7,8,9,10];
    $size_limit = 2;

    $i = 0;
    do {
        $limit = $i . "," . $size_limit;
        $i += $size_limit;
        $datas = array_slice($data, $limit, $size_limit);
        foreach ($datas as $value) {
            yield $value;
        }
    } while (count($datas) >= $size_limit);
}

foreach (getData() as $value) {
    echo $value . "\n";
}