# ETCD BOX

一个简单的ETCD可视化工具。

## About

之前用过两个ETCD可视化工具，[etcd-viewer](https://github.com/nikfoundas/etcd-viewer) 和 [etcdkeeper](https://github.com/evildecay/etcdkeeper) 。
etcd-viewer说是支持etcdv3，但是我死活连不上，不知道是不是我操作的问题。
个人觉得etcdkeeper比etcd-viewer好用，但是etcdkeeper有个问题，当etcd存储的数据量较大时，查询经常超时。
所以决定自己写个查询的小工具。

ETCD BOX仅支持window操作系统，并且存在很多问题，已知的就有图标显示错误等。

## Start

### Build

```
go build -o ETCDBox.exe -ldflags="-H windowsgui"
```

### Run

```
ETCDBox.exe
```

ETCDBox.exe启动前需要在同一目录下增加 `config.json` 配置文件

### Usage

就一个简单的查询功能，应该打开看到界面就会操作。

## Acknowledgements

感谢 [Walk](https://github.com/lxn/walk) 的 Window GUI 开发工具。

## License

MIT
