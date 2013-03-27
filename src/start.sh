#!/bin/bash
go build weixin.go
ps uax | grep weixin | awk 'NR==FNR{system("kill "$2) }'

sleep 10

nohup ./weixin >> stdout.log 2>&1 &