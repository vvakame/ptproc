# test for maprange

no skip
<!-- maprange:external.txt,test -->
```text
foobar
```
<!-- maprange.end -->

skip 1
<!-- maprange:file:"external.txt",name:"test",skip:1 -->
```text
foobar
```
<!-- maprange.end -->

skip 1 with no inline lines
<!-- maprange:file:"external.txt",name:"test",skip:1 -->
<!-- maprange.end -->

skip 10
<!-- maprange:file:"external.txt",name:"test",skip:10 -->
a
b
c
<!-- maprange.end -->
