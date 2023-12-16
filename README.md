> [!NOTE]
> 
> I retired as a macOS user on January 1st, 2021. After nearly a decade as a Mac user I'm now back with Linux on my desktop.
> 
> The one thing I miss from macOS is iTerm2, but since there is no iTerm2 on Linux this is the way it has to be. This also means the support for ssh2iterm2 will probably be limited to merging pull requests from @dependabot for as long as stuff keeps working.
> 
> I really have no clue whether there are any users of this besides myself, so please drop me a note in [#58](https://github.com/arnested/ssh2iterm2/issues/58) if you do use it. We'll figure out the future from there.

# Create iTerm2 dynamic profile from SSH config

Converts your `~/.ssh/config` to Dynamic profiles in iTerm2.

```shell
$ brew install arnested/ssh2iterm2/ssh2iterm2
```

By default it looks up your `Host` definitions in `~/.ssh/config`.

You can supply another location via the environment variable
`SSH2ITERM2_GLOB`.

I.e. set `SSH2ITERM2_GLOB=~/.ssh/config.d/*.conf` to run through all
`*.conf` files in `~/.ssh/config.d` and `SSH2ITERM2_GLOB=~/.ssh/**/*.conf` will run through all
`*.conf` files in all folders under `~/.ssh`.

The glob pattern should follow Gos [path/filepath patterns](https://pkg.go.dev/path/filepath#Match).

## Config file

Config will be read from `~/Library/Application
Support/ssh2iterm2.yaml` if the file exists.

The content of the config file could look like:

```yaml
glob: ~/.ssh/**/*.conf
ssh: /usr/local/bin/ssh
```

An alternate config file can be read using the `--config` option or
the `$SSH2ITERM2_CONFIG_FILE` environment variable.

## How to run

Just run the binary without any arguments in whatever directory you
like.

## The generated dynamic profile

The generated dynamic profile has some features/caveats (they suit me
well :-)

* The command calls `ssh` with an absolute path that is looked up when
  generating the dynamic profile. That is because iTerm2 doesn't have
  `/usr/local/bin` in its path and we would not be able to find a ssh
  installed by i.e. [Homebrew](https://brew.sh) otherwise.

  ```
  /usr/local/bin/ssh <host>
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

  This can be disabled with the option `--password-triggers=false`.

## Download

A compiled MacOS binary can be downloaded from [releases](https://github.com/arnested/ssh2iterm2/releases/latest).
