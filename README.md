# sshx

sshx is a convenient ssh tool, auto send passwords or execute commands like expect, effectively keep connections alive by simulating inputs, quickly log in by aliases.

## Installation

Prebuilt binaries for Linux and macOS can be downloaded from the [GitHub releases page](https://github.com/vj1024/sshx/releases).

You can also build from source code with `golang` installed:

``` bash
git clone git@github.com:vj1024/sshx.git
cd sshx
./build.sh
sudo mv sshx /usr/bin/
sshx localhost
```

## Configuration

Configuration file is optional.
The default file location is ~/.ssh/sshx.yaml.
It can also be modified through the environment variable 'SSHX_CONFIG' or the command line option '-c'.

``` yaml
servers:

- alias: [ test ]
  host: localhost 
  port: 22
  user: test_user
  password: 'test_password'

- alias: [ test1, user1 ]
  host: localhost 
  port: 2222
  user: root 
  expect:
  - match: '(P|p)assword:' # regular expression
    send: 'root_password'
  - match: '#'
    send: 'whoami'
    end: true
  idle_max_seconds: 600
  idle_send_string: '@'
```

## License

The code in this repository is released under the MIT License.
