# test for mapfile

no skip
<!-- mapfile:file:"external.txt" -->
hello, world!
<!-- mapfile.end -->

skip 1
<!-- mapfile:file:"external.txt",skip:1 -->
```text
hello, world!
```
<!-- mapfile.end -->

skip 1 with no inline lines
<!-- mapfile:file:"external.txt",skip:1 -->
hello, world!
<!-- mapfile.end -->

skip 10
<!-- mapfile:file:"external.txt",skip:10 -->
a
b
c
hello, world!
<!-- mapfile.end -->
