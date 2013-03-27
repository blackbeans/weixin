package main

import (
	"github.com/garyburd/go-mongo/mongo"
	"github.com/opesun/goquery"
	"log"
	"strconv"
)

const (
	targetUrl = "http://deliciouseveryday.diandian.com/page/"
)

func main() {

	pool := mongo.NewDialPool("localhost:27018", 1000)

	var i int = 1
	ch := make([]chan bool, 20)
	for ; i <= 20; i++ {
		ch[i-1] = make(chan bool)
		go func(i int, ch chan bool) {
			conn, _ := pool.Get()
			db := &mongo.Database{conn, "meishi", mongo.DefaultLastErrorCmd}
			coll := db.C("foods")
			data, err := goquery.ParseUrl(targetUrl + strconv.Itoa(i))
			if nil != err {
				log.Fatalln("response fail ,", err)
				ch <- false
				return
			}

			nodes := data.Find("#page-" + strconv.Itoa(i))
			size := nodes.Find("div.media").Length()
			for idx := 0; idx < size; idx++ {
				item := nodes.Find("div.media").Eq(idx)
				h2 := item.Find("h2")
				link := h2.Find("a")
				href := link.Attr("href")
				name := link.Attr("title")

				img := item.Find("img").Attr("src")
				// log.Println(name, "|", href, "|", img)
				if len(name) > 0 {
					err := coll.Insert(mongo.M{"name": name, "img_url": img, "link": href})
					log.Println("insert mongo|", err, "|", href)
				}
			}
			ch <- true
			log.Println(i)

		}(i, ch[i-1])
	}

	for i, val := range ch {
		<-val
		log.Println("end:", i)
	}
}
