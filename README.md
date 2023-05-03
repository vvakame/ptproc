# ptproc - Plain Text PROCessor

[review-preproc](https://github.com/kmuto/review/blob/master/doc/preproc.md) [(JA)](https://github.com/kmuto/review/blob/master/doc/preproc.ja.md) like text processor.

## Install

```shell
$ go install github.com/vvakame/ptproc/cmd/ptproc
```

## `mapfile` directive

`mapfile` directive embeds specified file.

```text
Hello, world!
mapfile:external.txt
mapfile.end
Good night, world.
```

```text
Good afternoon, world.
```

```text
Hello, world!
mapfile:external.txt
Good afternoon, world.
mapfile.end
Good night, world.
```

## `maprange` directive

`maprange` directive embeds segment of the specified file.

```text
Hello, world!
maprange:external.txt,targetB
maprange.end
Good night, world.
```

```text
foo
range:targetA
Good afternoon, world.
range.end
range:targetB
Good evening, world.
range.end
bar
```

```text
Hello, world!
maprange:external.txt,targetB
Good evening, world.
maprange.end
Good night, world.
```

## examples

```shell
$ ptproc ./_misc/testdata/**/testcase/test.md
```

```shell
$ docker run -v "$(pwd):/work" ghcr.io/vvakame/ptproc \
    --logLevel debug \
    --replace \
    --config ./_misc/config/default/testcase/ptproc.yaml \
    --glob "./_misc/testdata/*/*/testcase/test.md"
```
