## What?

Piper is a [https://patchbay.pub/](https://patchbay.pub/) inspired shell tool that lets you connect separate shells together over a NATS based middleware.

You can use it to build things like network aware clipboards, notifications from servers to your desktop, ad hoc assembled work queues for doing shell based work on many cores and more.

 * It has two basic modes of operation: multi consumer and multi producer or a work queue style multi producer to a load shared group of consumers
 * By default the publisher will block until there are consumers, by default it will give up after 1 hour, you can adjust this using `--timeout 5m` or by setting `PIPER_TIMEOUT=5m` for example
 * An, asynchronous mode that avoid above blocking, is support if you have NATS JetStream
 * In synchronous mode no data is stored, it's all ephemeral and the data is private to either your own NATS servers or your account on Synadia NGS (NATS as a Service). This means we won't be doing anything like serving web pages but with the shell utility scope I quite like it
 * It's secure your data can not be accessed by anyone else
 * Your data is compressed, NATS isn't great for large payloads but this will help a bit

## Multi Producer to Multi Consumer

The n:n mode means any one can send a notify and any listeners on the given channel will all get the message, here are a few use cases.

### Network aware speech synth

On your desktop(s) listen to messages, here using the OS X CLI tool `say` that reads out whatever it receives over your speakers:

```
while true
do
  piper listen say | xargs say
end
```

On any other machine:

```
$ ./longjob.sh ; piper say "long job completed"
```

Once `longjob.sh` completes your speakers will say it's done.

You can also `brew install terminal-notifier` and use that to get nice popups on your desktop with configurable titles, sound etc, for example instead of `xargs say` you'd use `terminal-notifier -title Piper -sound default`

## Network aware clipboard

On your desktop(s) run this:

```
while true
do
  piper listen clipboard | pbcopy
end
```

Now on any other machine you can send data to your clipboard quite easily:

```
$ ls | piper clipboard
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

## Asynchronous Mode

The examples above shows a blocking - synchronous - operation. While no persistence is needed it does mean your notifier is coupled to your listener and if either dies so does all progress.

Piper supports NATS JetStream (currently in preview) for persistence, in this case the notifier will publish data into JetStream and the listener will read from JetStream like a work queue.

If your NATS network has JetStream you can do `piper setup` to configure the required settings in JetStream and pass `-a` or set `PIPER_ASYNC=1` in your environment.  Once this is done Piper will be asynchronous.

## Setup

To use it you need a [NATS](https://nats.io) server, or you can sign up for a free [NGS](https://synadia.com/ngs) account.

To use your own NATS server set `PIPER_SERVER=your.nats.server:4222`.

If you need a NATS credential, such as with NGS or your own account enabled NATS, set `PIPER_CREDENTIAL=/path/to/your.creds` or place the same file in `~/.piper.creds`.

When in persistent mode data is kept for up to 24 hours or until any listener consumed it whichever comes first.

### Synadia NGS

[NGS](https://synadia.com/ngs) is a managed NATS network, it's global deployed in many regions and delivered as a utility, there is a generous free tier thats a really good fit for this tool.  Data is private within an account so other users would not be able to see your piper data, though if you wished you could arrange sharing between accounts.

Once you've signed up for an account you can create your own piper user limited to what piper needs like this:

```
$ nsc add user -a YOUR_ACCOUNT \
     --allow-pub "piper.>,_INBOX.>" \
     --allow-sub "piper.>,_INBOX.>" \
     --name piper
```

Just change `YOUR_ACCOUNT` to your own account name, then:

This will create `~/.nkeys/creds/synadia/YOUR_ACCOUNT/piper.creds ~/.piper.creds`, you'd copy this one file to any node that wants to use piper and NGS.

```
$ cp ~/.nkeys/creds/synadia/YOUR_ACCOUNT/piper.creds ~/.piper.creds
$ export PIPER_NGS=1
```

And you're good to go, you'll now connect to your nearest NGS server and the traffic will securely travel within your own account with no others being able to see it.

The subject names that piper makes are basically `piper.<name>`, so using the standard facilities in NGS `nsc` utility you could share piper data between different accounts securely.

## TODO?

 * Encryption of the data
 * Once there is a data store as a service on NGS I'd like to add features that can persist data into that

## Status?

Basic features work as I want them to work and I really like it

## Contact?

R.I.Pienaar | rip@devco.net | [@ripienaar](https://twitter.com/ripienaar)
