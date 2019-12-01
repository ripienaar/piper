## What?

Piper is a [https://patchbay.pub/](https://patchbay.pub/) inspired shell tool that lets you connect separate shells together.

Like patchbay it has two basic modes of operation, either multi consumer and multi producer or a pubsub style multi producer to a load shared group of consumers.

## Setup

To use it you need a [NATS](https://nats.io) server, or you can sign up for a free [NGS](https://synadia.com/ngs) account.

To use your own NATS server set `PIPER_SERVER=your.nats.server:4222`, to use NGS set `PIPE_SERVER=connect.ngs.global:4222`.

If you need a NATS credential, such as with NGS, set `PIPER_CREDENTIAL=/path/to/your.creds`.

## Multi Producer to Multi Consumer

On any server:

```
$ ./longjob.sh && echo done | piper notify xyz
```

On one of many desktops, they will all get the message:

```
$ piper listen xyz | notify-send "Job Done"
```

## Multi Producer to Load shared grouped Consumers

To demonstrate the load shared group we can resize a set of images across 2 shells, more useful utilities would be to fetch images from a s3 bucket and put smaller ones back, the requests would just be the file names or urls, but lets keep it simple:

On one shell run this:

```
#!/bin/bash

for filename in *.png
do
    echo $filename | piper notify resize
done
```

On four other shells run this:

```
#!/bin/bash

while true
do
  filename=$(piper listen resize --group)
  echo $filename
  convert -size 50%x50% "${filename}" "$(basename '$filename')-small.jpg"
done
```

## Status?

It's a bit of a toy right now, will work on it a bit more soon.

## Contact?

R.I.Pienaar | rip@devco.net | @ripienaar