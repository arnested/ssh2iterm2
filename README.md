# Create iTerm2 dynamic profile from SSH config

[![Build Status](https://travis-ci.org/arnested/ssh2iterm2.svg?branch=master)](https://travis-ci.org/arnested/ssh2iterm2)
[![release](https://img.shields.io/github/release/arnested/ssh2iterm2.svg)](https://github.com/arnested/ssh2iterm2/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/arnested/ssh2iterm2)](https://goreportcard.com/report/github.com/arnested/ssh2iterm2)

Converts your `~/.ssh/config` to Dynamic profiles in iTerm2.

By default it looks up your `Host` definitions in `~/.ssh/config`.

You can supply another location via the environment variable
`SSH2ITERM2_GLOB`.

I.e. set `SSH2ITERM2_GLOB=~/.ssh/config.d/*.conf` to run through all
`*.conf` files in `~/.ssh/config.d` and `SSH2ITERM2_GLOB=~/.ssh/**/*.conf` will run through all
`*.conf` files in all folders under `~/.ssh`.

The glob pattern should follow Gos [path/filepath patterns](https://golang.org/pkg/path/filepath/#Match).

## How to run

Just run the binary without any arguments in whatever directory you
like.

## The generated dynamic profile

The generated dynamic profile has some features/caveats (they suit me
well :-)

* The command is not a direct call to `ssh`. That is because iTerm2
  doesn't have `/usr/local/bin` in its path. Instead we wrap it in a
  call to `sh`:

  ```
  sh -c 'PATH=/usr/local/bin:$PATH ssh <host>'
  ```

* We add the host as a badge

* If the the filename where the Host is defined is _not_ `config` we
  use the filename as a tag on the profile (extension removed from
  file, preprending digits followed by underscore removed).

  This way you can group your Hosts.

  I.e. all Hosts defined in `20_production.conf` will get a
  "production" tag.

* A trigger that opens iTerm2s password manager is added on the
  regular expression `\\[sudo\\] password for`.

  The password manager will get the host name as parameter.

## Download

A compiled MacOS binary can be downloaded from [releases](https://github.com/arnested/ssh2iterm2/releases/latest).
