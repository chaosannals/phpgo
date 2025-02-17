<?php

$content = file_get_contents("demo.txt");
$r = unserialize($content);
var_export($r);
