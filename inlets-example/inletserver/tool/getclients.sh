# 查询inlet client列表 ,拿到 x-clients-id ，然后请求
curl http://127.0.0.1:28815/inlet/clients 
curl -X POST -d '{"cmd":"set"}'  -H "x-clients-id:c0e84cf2df4343a3b382de9f39528baa"  http://127.0.0.1:28815/inlet/cmd/echo