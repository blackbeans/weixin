package main

import (
	"github.com/garyburd/go-mongo/mongo"
	"github.com/opesun/goquery"
	"log"
	"strconv"
)

const (
	targetUrl = "http://www.meishij.net/list.php?lm=43&page="
)

func main() {

	pool := mongo.NewDialPool("localhost:27018", 1000)

	var i int = 1
	chs := make([]chan bool, 50)
	hasMore := true
	for ; i <= 50; i++ {
		ch := make(chan bool)

		chs[i-1] = ch

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

			nodes := data.Find("div.lp_result_list")
			size := nodes.Find("li").Length()
			if size <= 0 {
				hasMore = false
			}
			for idx := 0; idx < size; idx++ {
				item := nodes.Find("li").Eq(idx)

				link := item.Find("a")
				href := link.Attr("href")
				name := link.Attr("title")
				img := item.Find("img").Attr("src")
				// log.Println(name, "|", href, "|", img)
				if len(name) > 0 {

					err := coll.Upsert(mongo.M{"name": name}, mongo.M{"name": name, "img_url": img, "link": href})

					// log.Println("insert mongo|", err, "|", href)
					log.Println("err", err, "name:", name, "link:", link, "href:", href, "img", img)
				}
			}
			ch <- true
			log.Println(i)

		}(i, ch)
	}

	for i, val := range chs {
		<-val
		log.Println("end:", i)
	}
}
