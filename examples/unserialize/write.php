<?php

namespace Demo\Ns;

class DemoE {
    public $c;
    private $d;

    public function __construct($c, $d) {
        $this->c = $c;
        $this->d = $d;
    }

    public function getD() {
        return $this->d;
    }

    public function setD($v) {
        $this->d = $v;
    }
}

class DemoA {
    private $a;
    public $b;
    public $m;
    public $list;
    private $e;

    public function __construct($a, $b) {
        $this->a = $a;
        $this->b = $b;
        $this->m = [ 'd' => 123, 'c' => 'f' ];
        $this->list = [1,2,3,4];
        $this->e = new DemoE("1234", [12345,567,3434]);
    }

    public function getA() {
        return $this->a;
    }

    public function setA($v) {
        $this->a = $v;
    }

    public function getE() {
        return $this->e;
    }
}

$da = new DemoA("123", 3434);
$daTxt = serialize($da);

$daFile = "demo.txt";
file_put_contents($daFile,  $daTxt, LOCK_EX);


