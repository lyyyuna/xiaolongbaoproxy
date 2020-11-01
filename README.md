# Xiaolongbao Proxy

[Xiaolongbao](https://en.wikipedia.org/wiki/Xiaolongbao) is a type of Chinese steamed bun from Jiangsu province, especially associated with Wuxi.

## Main features

* Basic Http proxy for Http/Https
* Support Mitm mode for both Http and Https
    1. you can change to your own root CA
    2. you can add hook function to recrod Http/Https' content

## Quick start

### Help

```
Start a HTTP/S proxy

Usage:
  xiaolongbaoproxy [command]

Available Commands:
  basic       Start a basic http proxy
  help        Help about any command
  mitm        Start a mitm http proxy
  version     Print the version of the xiaolongbao proxy

Flags:
  -h, --help   help for xiaolongbaoproxy

Use "xiaolongbaoproxy [command] --help" for more information about a command.
```

### Start a basic proxy

```
xiaolongbaoproxy basic --server "127.0.0.1" --port 8081
```

### Start a mitm proxy

```
xiaolongbaoproxy mitm
```

## Customize

Refer to cmd folders, add hook functions in your own cmds.