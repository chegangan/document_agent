
docker save -o document-agent-usercenter-rpc-1.0.tar chegangan/document-agent-usercenter-rpc:1.0
docker save -o document-agent-usercenter-api-1.0.tar chegangan/document-agent-usercenter-api:1.0

docker load -i document-agent-usercenter-api-1.0.tar
docker load -i document-agent-usercenter-rpc-1.0.tar