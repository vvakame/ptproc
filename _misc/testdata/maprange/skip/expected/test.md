# test for maprange

no skip
<!-- maprange:external.txt,test -->
good afternoon, world!
<!-- maprange.end -->

skip 1
<!-- maprange:file:"external.txt",name:"test",skip:1 -->
```text
good afternoon, world!
```
<!-- maprange.end -->

skip 1 with no inline lines
<!-- maprange:file:"external.txt",name:"test",skip:1 -->
good afternoon, world!
<!-- maprange.end -->

skip 10
<!-- maprange:file:"external.txt",name:"test",skip:10 -->
a
b
c
good afternoon, world!
<!-- maprange.end -->
