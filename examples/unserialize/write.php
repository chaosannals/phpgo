<?php

namespace Demo\Ns;

class DemoA {
    private $a;
    public $b;
    public $m;
    public $list;

    public function __construct($a, $b) {
        $this->a = $a;
        $this->b = $b;
        $this->m = [ 'd' => 123, 'c' => 'f' ];
        $this->list = [1,2,3,4];
    }

    public function getA() {
        return $this->a;
    }

    public function setA($v) {
        $this->a = $v;
    }
}

$da = new DemoA("123", 3434);
$daTxt = serialize($da);

$daFile = "demo.txt";
file_put_contents($daFile,  $daTxt, LOCK_EX);


