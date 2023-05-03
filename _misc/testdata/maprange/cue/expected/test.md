# test for maprange

plain text
<!-- maprange:external.txt,test -->
good afternoon, world!
<!-- maprange.end -->

cue with structure
<!-- maprange:file:"external.txt",name:"test" -->
good afternoon, world!
<!-- maprange.end -->

cue with JSON syntax
<!-- maprange:{"file":"external.txt","name":"test"} -->
good afternoon, world!
<!-- maprange.end -->

cue with string literal
<!-- maprange:"external.txt,test" -->
good afternoon, world!
<!-- maprange.end -->
