<?php

$typeNull = NULL;
$typeBool = true;
$typeInt = 3;
$typeFloat = 98.9;
$typeString = "Hello Wrold";
$typeArray = array(
    1,
    2,
);
class Study
{
}
$typeObject = new Study();

$typeFunc = function () {
    return 1;
};
$typeResource = STDIN;

// 使用 var_dump 打类型与值
var_dump($typeNull);
var_dump($typeBool);
var_dump($typeInt);
var_dump($typeFloat);
var_dump($typeString);
var_dump($typeArray);
var_dump($typeObject);
var_dump($typeFunc);
var_dump($typeResource);

echo "------------------------" . PHP_EOL;

// 使用 gettype 获取各类型的名称
printf("typeNull type is %s\n", gettype($typeNull));
printf("typeBool type is %s\n", gettype($typeBool));
printf("typeInt type is %s\n", gettype($typeInt));
printf("typeFloat type is %s\n", gettype($typeFloat));
printf("typeString type is %s\n", gettype($typeString));
printf("typeArray type is %s\n", gettype($typeArray));
printf("typeObject type is %s\n", gettype($typeObject));
printf("typeFunc type is %s\n", gettype($typeFunc));
printf("typeResource type is %s\n", gettype($typeResource));

echo "------------------------" . PHP_EOL;

// 使用 is_type 判断类型
if (is_null($typeNull)) {
    echo "typeNull is null" . PHP_EOL;
}
if (is_bool($typeBool)) {
    echo "typeBool is bool" . PHP_EOL;
}
if (is_int($typeInt)) {
    echo "typeInt is int" . PHP_EOL;
}
if (is_float($typeFloat)) {
    echo "typeFloat is float" . PHP_EOL;
}
if (is_string($typeString)) {
    echo "typeString is string" . PHP_EOL;
}
if (is_array($typeArray)) {
    echo "typeArray is array" . PHP_EOL;
}
if (is_object($typeObject)) {
    echo "typeObject is object" . PHP_EOL;
}
if (is_callable($typeFunc)) {
    echo "typeFunc is callable" . PHP_EOL;
}
if (is_resource($typeResource)) {
    echo "typeResource is resource" . PHP_EOL;
}
