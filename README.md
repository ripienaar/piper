## What?

Piper is a [https://patchbay.pub/](https://patchbay.pub/) inspired shell tool that lets you connect separate shells together.

Like patchbay it has two basic modes of operation, either multi consumer and multi producer or a work queue style multi producer to a load shared group of consumers.

Like patchbay the publisher will block until there are consumers.

## Setup

To use it you need a [NATS](https://nats.io) server, or you can sign up for a free [NGS](https://synadia.com/ngs) account.

To use your own NATS server set `PIPER_SERVER=your.nats.server:4222`.

If you need a NATS credential, such as with NGS, set `PIPER_CREDENTIAL=/path/to/your.creds`.

### NGS

If you use a [NGS](https://synadia.com/ngs) account - free tier is fine for most uses of this tool - you can create a user just for piper like this:

```
$ nsc add user -a YOUR_ACCOUNT --allow-pub "piper.>,_INBOX.>" --allow-sub "piper.>,_INBOX.>" --name piper
```

Just change `YOUR_ACCOUNT` to your own account name, then:

```
$ export PIPER_SERVERS=connect.ngs.global:4222
$ export PIPER_CREDENTIALS=~/.nkeys/creds/synadia/YOUR_ACCOUNT/piper.creds
```

And you're good to go, you'll now connect to your nearest NGS server and the traffic will securely travel within your own account with no others being able to see it.

The subject names that piper makes are basically `piper.<name>`, so using the standard facilities in NGS `nsc` utility you could share piper data between different accounts securely.

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

To demonstrate the load shared group we can resize a set of png images across 2 shells, more useful utilities would be to fetch images from a s3 bucket and put smaller ones back, the requests would just be the file names or urls, but lets keep it simple:

On one shell run this:

```
#!/bin/bash

for filename in *.png
do
    echo $filename | piper notify resize
done
```

On 2 other shells run this:

```
#!/bin/bash

while true
do
  filename=$(piper listen resize --group)
  base=$(basename "${filename}" .png)
  echo $filename
  convert -size 50%x50% "${filename}" "${base}-small.jpg"
done
```

Your images will now be resized using max 2 cores.

## Status?

It's a bit of a toy right now, will work on it a bit more soon.

## Contact?

R.I.Pienaar | rip@devco.net | @ripienaar