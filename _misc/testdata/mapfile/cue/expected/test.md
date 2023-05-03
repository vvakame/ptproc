# test for mapfile

plain text
<!-- mapfile:external.txt -->
hello, world!
<!-- mapfile.end -->

cue with structure
<!-- mapfile:file:"external.txt" -->
hello, world!
<!-- mapfile.end -->

cue with JSON syntax
<!-- mapfile:{"file":"external.txt"} -->
hello, world!
<!-- mapfile.end -->

cue with string literal
<!-- mapfile:"external.txt" -->
hello, world!
<!-- mapfile.end -->
