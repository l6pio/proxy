#!/usr/bin/env bash
export http_proxy=http://localhost:3210
export https_proxy=http://localhost:3210
#echo ">>> HTTP >>>"
#curl http://www.baidu.com -H "Host:www.baidu.com" -v
echo ">>> HTTPS >>>"
curl https://xueshu.baidu.com/usercenter/paper/show?paperid=80d45d320348b4f7293181b60c0e5de9 -X GET -v --insecure
#curl https://www.baidu.com -X GET -H "Host:www.baidu.com" -v
