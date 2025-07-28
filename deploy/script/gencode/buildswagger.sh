#!/bin/bash

# 检查参数数量
if [ $# -ne 2 ]; then
  echo "用法: $0 <服务名> <api文件名>"
  echo "示例: $0 usercenter usercenter.api"
  exit 1
fi

SERVICE_NAME=$1
API_FILE=$2

goctl api swagger --api ../../../app/${SERVICE_NAME}/cmd/api/desc/${API_FILE} --dir ../../swagger