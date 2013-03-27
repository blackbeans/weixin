#!/bin/bash
go build meishi.go
ps uax | grep weixin | awk 'NR==FNR{system("kill "$2) }'

sleep 10

nohup ./meishi >> stdout.log 2>&1 &