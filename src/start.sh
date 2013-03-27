#!/bin/bash
go build weixin.go
ps uax | grep weixin | awk 'NR==FNR{system("kill "$2) }'

sleep 5

nohup ./$GOPATH/bin/weixin >> stdout.log 2>&1 &