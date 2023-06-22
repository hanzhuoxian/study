<?php

use Yaf\Application;

define("APP_PATH",  dirname(realpath(dirname(__FILE__)))); /* 指向public的上一级 */
$app  = new Application(APP_PATH . "/conf/application.ini");
$app->run();

