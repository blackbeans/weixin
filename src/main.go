package main

import "entry"

func main() {
	var ins entry.RespMessage
	txt := entry.TxtResponse{}
	pic := entry.PicResponse{}
	ch := make(chan interface{})
	ch <- txt
	defer close(ch)
	ins = txt
}
