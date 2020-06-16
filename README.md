# paxy

`paxy` is an HTTP proxy with support for PAC files.

The main use case is to expose a single HTTP proxy which follows a set of rules
to select different proxies depending on the hostname of the connection.

## Installation

```bash
go get -u -v github.com/robbiemcmichael/paxy
```

## Usage

```bash
paxy path/to/proxy.pac
```

Then configure your software to use the proxy. For example:

```bash
export http_proxy=http://127.0.0.1:8228
export https_proxy=http://127.0.0.1:8228
curl https://github.com
```

### Options

```
Usage: paxy [options] pac_file ...
  -p int
    	The port on which the server listens (default 8228)
```
