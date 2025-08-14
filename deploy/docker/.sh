
docker save -o document-agent-usercenter-rpc.tar document-agent/usercenter-rpc
docker save -o document-agent-usercenter-api.tar document-agent/usercenter-api
docker save -o document-agent-llmcenter-rpc.tar document-agent/llmcenter-rpc
docker save -o document-agent-llmcenter-api.tar document-agent/llmcenter-api

docker load -i document-agent-usercenter-api.tar
docker load -i document-agent-usercenter-rpc.tar
docker load -i document-agent-llmcenter-api.tar
docker load -i document-agent-llmcenter-rpc.tar


SSH，访问10.16.23.106，22端口
用户名:tzb,密码：tzb123456

[Interface]
Address = 10.0.13.4/24
PrivateKey = CAAexGRb4nNe/CLjPIYu4TaWAltS0sWV+9Qwk4v6CnQ=

[Peer]
PublicKey = 0NJ3xphTj7HC3VmVcXUNciFad2u8nV1ZJyglGoYUbwc=
Endpoint = 47.98.215.181:51822
AllowedIPs = 10.0.13.0/24,10.16.23.106/24,192.168.1.111/24


sudo wg-quick up wg0
ssh tzb@10.16.23.106

http://47.98.215.181:8010/