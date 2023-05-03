# test for mapfile

no skip
<!-- mapfile:file:"external.txt" -->
```text
foobar
```
<!-- mapfile.end -->

skip 1
<!-- mapfile:file:"external.txt",skip:1 -->
```text
foobar
```
<!-- mapfile.end -->

skip 1 with no inline lines
<!-- mapfile:file:"external.txt",skip:1 -->
<!-- mapfile.end -->

skip 10
<!-- mapfile:file:"external.txt",skip:10 -->
a
b
c
<!-- mapfile.end -->
