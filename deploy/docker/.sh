
docker save -o document-agent-usercenter-rpc.tar document-agent/usercenter-rpc
docker save -o document-agent-usercenter-api.tar document-agent/usercenter-api
docker save -o document-agent-llmcenter-rpc.tar document-agent/llmcenter-rpc
docker save -o document-agent-llmcenter-api.tar document-agent/llmcenter-api

rm document-agent-llmcenter-rpc.tar
rm -rf ./deploy
rm docker-compose.yml


# 2. 单独上传 docker-compose.yml
scp -i ~/.ssh/id_rsa ~/myspace/code/golang/document_agent/docker-compose.yml tzb@10.16.23.106:~/back/

# 3. 单独上传每个 tar 包
scp -i ~/.ssh/id_rsa ~/myspace/code/golang/document_agent/document-agent-llmcenter-api.tar tzb@10.16.23.106:~/back/
scp -i ~/.ssh/id_rsa ~/myspace/code/golang/document_agent/document-agent-llmcenter-rpc.tar tzb@10.16.23.106:~/back/
scp -i ~/.ssh/id_rsa ~/myspace/code/golang/document_agent/document-agent-usercenter-api.tar tzb@10.16.23.106:~/back/
scp -i ~/.ssh/id_rsa ~/myspace/code/golang/document_agent/document-agent-usercenter-rpc.tar tzb@10.16.23.106:~/back/

# 4. （可选）上传 deploy 文件夹（增量同步）
rsync -avz -e "ssh -i ~/.ssh/id_rsa" ~/myspace/code/golang/document_agent/deploy/ tzb@10.16.23.106:~/back/deploy/

ssh -i ~/.ssh/id_rsa tzb@10.16.23.106 bash -s <<'EOF'
set -e
cd ~/back || { echo "目录 ~/back 不存在"; exit 1; }
echo "Running docker compose down..."
docker compose -f docker-compose.yml down --rmi local -v --remove-orphans || true

echo "Removing images by full name..."
docker rmi -f \
  document-agent/llmcenter-api:latest \
  document-agent/usercenter-api:latest \
  document-agent/usercenter-rpc:latest \
  document-agent/llmcenter-rpc:latest || true

echo "Remaining document-agent images:"
docker images | grep '^document-agent/' || echo "none"
EOF

docker exec -it llmcenter-rpc /bin/bash

docker load -i document-agent-usercenter-api.tar
docker load -i document-agent-usercenter-rpc.tar
docker load -i document-agent-llmcenter-api.tar
docker load -i document-agent-llmcenter-rpc.tar
docker compose up -d 

sudo wg-quick up wg0
sudo wg-quick down wg0
ssh tzb@10.16.23.106

http://47.98.215.181:8010/

[Interface]
Address = 10.0.13.4/24
PrivateKey = CAAexGRb4nNe/CLjPIYu4TaWAltS0sWV+9Qwk4v6CnQ=

[Peer]
PublicKey = 0NJ3xphTj7HC3VmVcXUNciFad2u8nV1ZJyglGoYUbwc=
Endpoint = 47.98.215.181:51822
AllowedIPs = 10.0.13.0/24,10.16.23.106/24,192.168.1.111/24


# 使用hey 压测数据
# -n 2000: 总共发送 2000 个请求。-c 100: 模拟 100 个并发用户。
hey -n 2000 -c 100 -m POST \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODQ4MDc4MzQsImlhdCI6MTc1MzI3MTgzNCwiand0VXNlcklkIjoxfQ.lLUGgqAeF2VDdfJEBHe8r-Mzv8da_pURz0251osJtA8" \
-H "Content-Type: application/json" \
-d '{
    "conversation_id": "01K627769R3S6Z8QZKZT29F84E",
    "message_id": "01K627769R3S6Z8QZKZT29F84E", 
    "prompt": "请帮我把时间改为2025.2.11", 
    "use_knowledge_base": false,
    "knowledge_base_id": ""
}' \
"http://localhost:8010/llmcenter/v1/chat/edit"


压测结果：第一个是加了redis缓存的，第二个是没加redis缓存的
chegan@workplace:~$ hey -n 2000 -c 100 -m POST \
-H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODQ4MDc4MzQsImlhdCI6MTc1MzI3MTgzNCwiand0VXNlcklkIjoxfQ.lLUGgqAeF2VDdfJEBHe8r-Mzv8da_pURz0251osJtA8" \
-H "Content-Type: application/json" \
-d '{
    "conversation_id": "01K627769R3S6Z8QZKZT29F84E",
    "message_id": "01K627769R3S6Z8QZKZT29F84E", 
    "prompt": "请帮我把时间改为2025.2.11", 
    "use_knowledge_base": false,
    "knowledge_base_id": ""
}' \
"http://localhost:8010/llmcenter/v1/chat/edit"

Summary:
  Total:	2.4440 secs
  Slowest:	0.7678 secs
  Fastest:	0.0129 secs
  Average:	0.0932 secs
  Requests/sec:	818.3152
  

Response time histogram:
  0.013 [1]	|
  0.088 [1357]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.164 [480]	|■■■■■■■■■■■■■■
  0.239 [126]	|■■■■
  0.315 [9]	|
  0.390 [0]	|
  0.466 [0]	|
  0.541 [0]	|
  0.617 [8]	|
  0.692 [0]	|
  0.768 [19]	|■


Latency distribution:
  10% in 0.0465 secs
  25% in 0.0579 secs
  50% in 0.0739 secs
  75% in 0.0987 secs
  90% in 0.1495 secs
  95% in 0.1918 secs
  99% in 0.6020 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0003 secs, 0.0129 secs, 0.7678 secs
  DNS-lookup:	0.0001 secs, 0.0000 secs, 0.0084 secs
  req write:	0.0001 secs, 0.0000 secs, 0.0059 secs
  resp wait:	0.0067 secs, 0.0007 secs, 0.5811 secs
  resp read:	0.0859 secs, 0.0090 secs, 0.7656 secs

Status code distribution:
  [200]	2000 responses



chegan@workplace:~$ hey -n 2000 -c 100 -m POST -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODQ4MDc4MzQsImlhdCI6MTc1MzI3MTgzNCwiand0VXNlcklkIjoxfQ.lLUGgqAeF2VDdfJEBHe8r-Mzv8da_pURz0251osJtA8" -H "Content-Type: application/json" -d '{
    "conversation_id": "01K627769R3S6Z8QZKZT29F84E",
    "message_id": "01K627769R3S6Z8QZKZT29F84E", 
    "prompt": "请帮我把时间改为2025.2.11", 
    "use_knowledge_base": false,
    "knowledge_base_id": ""
}' "http://localhost:8010/llmcenter/v1/chat/edit"

Summary:
  Total:	4.0417 secs
  Slowest:	2.6813 secs
  Fastest:	0.0090 secs
  Average:	0.1991 secs
  Requests/sec:	494.8377
  

Response time histogram:
  0.009 [1]	|
  0.276 [1899]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.543 [0]	|
  0.811 [0]	|
  1.078 [0]	|
  1.345 [0]	|
  1.612 [0]	|
  1.880 [0]	|
  2.147 [0]	|
  2.414 [0]	|
  2.681 [100]	|■■


Latency distribution:
  10% in 0.0461 secs
  25% in 0.0563 secs
  50% in 0.0702 secs
  75% in 0.0901 secs
  90% in 0.1197 secs
  95% in 2.5236 secs
  99% in 2.6278 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0004 secs, 0.0090 secs, 2.6813 secs
  DNS-lookup:	0.0001 secs, 0.0000 secs, 0.0089 secs
  req write:	0.0001 secs, 0.0000 secs, 0.0045 secs
  resp wait:	0.0055 secs, 0.0007 secs, 0.0444 secs
  resp read:	0.1930 secs, 0.0080 secs, 2.6626 secs

Status code distribution:
  [200]	2000 responses



chegan@workplace:~$ 



分析：
1. 吞吐量 (QPS) 大幅提升
QPS 从 494 提升到了 818，增加了 65.4%。
这意味着在同一时间内，你的服务器现在能够处理比以前多 65% 的用户请求。这是系统承载能力质的飞跃，直接证明了缓存极大地减轻了数据库的压力，将原本消耗在 I/O 等待上的 CPU 资源释放出来处理更多的业务逻辑。

2. 平均响应时间显著降低
平均延迟从 199.1ms 降低到了 93.2ms，减少了 53.2%。
这代表着用户的平均等待时间缩短了一半以上。对于用户体验来说，这是一个非常显著的改善，应用的响应变得更加“丝滑”了。

3. 系统稳定性 (P99 延迟) 质的飞跃
这是本次优化最亮眼的地方！
P99 延迟代表了 99% 的请求所花费的时间。优化前，P99 高达 2.6 秒，这是一个非常危险的信号。它说明在并发压力下，你的数据库出现了严重的瓶颈，导致部分用户的请求被长时间阻塞，体验极差，甚至可能导致服务雪崩。

而优化后，P99 延迟降低到了 602ms，降幅高达 77.1%！虽然这个值仍然比平均延迟高，但它表明系统在高并发下的表现极其稳定，绝大多数用户的请求都能在可接受的时间内得到响应。你成功地消除了系统的主要性能瓶颈，极大地提升了系统的健壮性和可靠性。