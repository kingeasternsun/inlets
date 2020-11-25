this example shows how to use inlets to visit the undirect server throuth proxy server(inlets server)

这个样例展示了如果利用inlets来访问无法直接访问的内部server，通过inlets构建一个代理server间接访问。

- server目录下为inlets-server,用来管理多个inlets-client
- client目录下为inlets-client,提供的api接口只能通过server进行访问
