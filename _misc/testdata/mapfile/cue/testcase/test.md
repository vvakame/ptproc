# test for mapfile

plain text
<!-- mapfile:external.txt -->
<!-- mapfile.end -->

cue with structure
<!-- mapfile:file:"external.txt" -->
<!-- mapfile.end -->

cue with JSON syntax
<!-- mapfile:{"file":"external.txt"} -->
<!-- mapfile.end -->

cue with string literal
<!-- mapfile:"external.txt" -->
<!-- mapfile.end -->
