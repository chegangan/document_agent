
docker save -o document-agent-usercenter-rpc.tar document-agent/usercenter-rpc
docker save -o document-agent-usercenter-api.tar document-agent/usercenter-api
docker save -o document-agent-llmcenter-rpc.tar document-agent/llmcenter-rpc
docker save -o document-agent-llmcenter-api.tar document-agent/llmcenter-api

docker load -i document-agent-usercenter-api.tar
docker load -i document-agent-usercenter-rpc.tar
docker load -i document-agent-llmcenter-api.tar
docker load -i document-agent-llmcenter-rpc.tar