<?php

namespace Singleton;

class Singleton
{

    private static $instance;

    private function __construct()
    {
        echo "Init " . __METHOD__ . PHP_EOL;
    }

    public static function getInstance()
    {
        if (is_null(self::$instance)) {
            self::$instance = new self();
        }
        return self::$instance;
    }

    private function __clone()
    {
    }
}

$singleton = Singleton::getInstance();
