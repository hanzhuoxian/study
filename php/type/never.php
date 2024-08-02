<?php

function foo(): never
{
    compact('a');
    new Exception();
}

foo();
