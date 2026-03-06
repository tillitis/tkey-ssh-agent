tkey-ssh-agent(1)

# NAME

tkey-ssh-agent – An SSH agent backed by Tillitis TKey

# SYNOPSIS

*tkey-ssh-agent* -h | --help

*tkey-ssh-agent* --version

*tkey-ssh-agent* -p | --show-pubkey

*tkey-ssh-agent* -L | --list-ports

*tkey-ssh-agent* [-a | --agent-path path] [--force-full-uss] [-p | --show-pubkey] [--pinentry command] [--port path] [--speed bit_speed] [--uss] [--uss-file path]

# DESCRIPTION

*tkey-ssh-agent* is an alternative SSH agent backed by an Ed25519
signer device-app which it uploads to the Tillitis TKey, a small
computer in the form of a USB stick.

It works as an OpenSSH-compatible agent for all SSH programs,
supporting a necessary subset of the OpenSSH agent protocol. You can
use it to login to other systems or to sign Git commits, for example.
Your ephemeral private key never leaves the TKey.

The act of uploading the signer app, with an optional User Supplied
Secret, creates a new unique, stable but ephemeral identity for that
specific combination of TKey, signer app binary, and USS.

The options are as follows:

*-L | --list-ports*

	List possible serial ports to use with *--port* and exit.

*-a | --agent-path path*

	Bind the agent to the UNIX-domain socket at path.

*--force-full-uss*

	Force sending a 32 byte USS digest. For backwards compatibility
	the default is sending 31 bytes of the computed digest and a zero.

*--help*

	Output help text and exit.

*-p | --show-pubkey*

	Extract the ssh-ed25519 public key from the TKey and exit.

*--pinentry command*

	Specify pinentry command for use by --uss. The default is found by
	looking in your *gpg-agent.conf* for pinentry-program. If this is not
	found, the *pinentry(1)* command is used.

*--port path*

	Set serial port device path. If this is not set, auto-detection will
	be attempted.

*--speed bit_speed*

	Set serial port speed in bits per second. Default is 62500 b/s.

*--uss*

	Interactively ask for a secret to be hashed as the User Supplied
	Secret. The USS is loaded onto the TKey along with the signer. A
	different USS results in a different pair of SSH public/private keys.

*--uss-file path*

	Read a secret from a file path and hash its contents as the User
	Supplied Secret to be mixed into the TKey identity. Use '-' (dash) to
	read from stdin. The full contents are hashed unmodified (i.e.
	newlines are not stripped).

*--version*

	Output version information.

## User Supplied Secret

You are encouraged to run *tkey-ssh-agent* with *--uss*, meaning that
when the signer app is loaded onto the TKey, a User Supplied Secret
(USS) is loaded as well. In short, changing the USS results in a
different SSH ed25519 key, a new identity. The USS is input
interactively using a *pinentry(1)* command, see the *--uss* option
for more information. If not run with *--uss*, the TKey/signer
combination will have a single identity.

## systemd-based systems

With the source code we provide a systemd unit file that can be used
to automatically start the tkey-ssh-agent when a user logs in. If this
unit file is installed (for instance with a package for your operating
system) you can run the following (as your own user) to start the
agent right now and enable it for future logins:

```
$ systemctl --user enable --now tkey-ssh-agent
```

# ENVIRONMENT

To make *ssh(1)*, *ssh-add(1)* and other tools use *tkey-ssh-agent*
you must set the *SSH_AUTH_SOCK* environment variable for them, so
they can find and communicate with the SSH agent. For your current
shell this can be done like this (bash/zsh):

```
export SSH_AUTH_SOCK="$XDG_RUNTIME_DIR/tkey-ssh-agent/sock"
```

This line can also be added to the startup file for your shell, e.g.
in .bashrc. This would make all tools that honour SSH_AUTH_SOCK use
tkey-ssh-agent.

With SSH_AUTH_SOCK correctly set you can see the current SSH ed25519
public key by running:

```
$ ssh-add -L
```

# FILES

*tkey-ssh-agent* does not have a configuration file.

You might, however, want to configure ssh(1) to use a specific SSH agent
("IdentityAgent") depending on the host you want to access. Add the
following to ~/.ssh/config to make it use tkey-ssh-agent when connecting
to "example.com":

```
Host example.com
  IdentityAgent ${XDG_RUNTIME_DIR}/tkey-ssh-agent/sock
```

Or use tkey-ssh-agent for all hosts except "example.com":

```
Host example.com
  IdentityAgent $SSH_AUTH_SOCK

Host *
  IdentityAgent ${XDG_RUNTIME_DIR}/tkey-ssh-agent/sock
```

# EXAMPLES

Running manually against a TKey with automatic port detection and
interactively ask for the User Supplied Secret:

```
$ tkey-ssh-agent -a ./agent.sock --uss
```

Running manually against qemu (look when qemu is starting what device it
uses) and interactively ask for the User Supplied Secret:

```
$ tkey-ssh-agent -a ./agent.sock --port /dev/pts/1 --uss
```

Ask the agent about the TKey public key:

```
$ SSH_AUTH_SOCK=./agent.sock ssh-add -L
```

Login to localhost using the agent (copy the public key to
*~/.ssh/authorized_key* first):

```
$ SSH_AUTH_SOCK=./agent.sock ssh -F /dev/null localhost
```

# SEE ALSO

*ssh(1)* *ssh-add(1)* *ssh-agent(1)*

# STANDARDS

*tkey-ssh-agent* attempts to follow a subset of the OpenSSH Agent
protocol. It doesn't do anything on calls to add/remove keys, or to
lock/unlock it with a passphrase.

# AUTHORS

Tillitis AB, https://tillitis.se/

# CAVEATS

The *tkey-ssh-agent* only connects to the TKey when an SSH agent
operation is requested (e.g. when you attempt to log in somewhere or
request the public key) and disconnects soon afterwards. If the signer
app is not already running on the TKey it is first uploaded to the
TKey and started.

This means that it will only ask for the User Supplied Secret (if
started using the *--uss* flag) when the agent is actually requested
to do something for the first time, not when the TKey is inserted, as
the user perhaps expected. The reason is that the tkey-ssh-agent
shouldn't hog the device and let other client apps also be able to
speak to it.

