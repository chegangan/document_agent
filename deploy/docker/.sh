
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


SSH，访问10.16.23.106，22端口
用户名:tzb,密码：tzb123456

[Interface]
Address = 10.0.13.4/24
PrivateKey = CAAexGRb4nNe/CLjPIYu4TaWAltS0sWV+9Qwk4v6CnQ=

[Peer]
PublicKey = 0NJ3xphTj7HC3VmVcXUNciFad2u8nV1ZJyglGoYUbwc=
Endpoint = 47.98.215.181:51822
AllowedIPs = 10.0.13.0/24,10.16.23.106/24,192.168.1.111/24


