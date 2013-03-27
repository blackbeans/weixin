package main

import (
	"crypto/sha1"
	"encoding/xml"
	"entry"
	"fmt"
	"github.com/garyburd/go-mongo/mongo"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"time"
)

const (
	basiurl    = "http://redis.io/commands/"
	forwardurl = "http://localhost:8080/"
	token      = "betago"
	welcome    = "欢迎使用美食助手应用，愿你在这里发现生活真正的意义~<br/>业务合作联系:blackbeans.zc@gmail.com"
)

var pool *mongo.Pool

func init() {
	pool = mongo.NewDialPool("localhost:27018", 1000)
}

func main() {
	http.HandleFunc("/weixin", WexinHandler)

	http.HandleFunc("/forward", ForwardHandler)
	http.ListenAndServe(":80", nil)
}

func ForwardHandler(wr http.ResponseWriter, req *http.Request) {
	link, err := url.Parse(forwardurl)
	if nil != err {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(link)
	proxy.ServeHTTP(wr, req)

}

func weixinValid(resp http.ResponseWriter, req *http.Request) {
	signature := req.FormValue("signature")
	timestamp := req.FormValue("timestamp")
	nonce := req.FormValue("nonce")
	echostr := req.FormValue("echostr")

	vali := func() string {
		strs := sort.StringSlice{token, timestamp, nonce}
		sort.Strings(strs)
		str := ""
		for _, s := range strs {
			str += s
		}

		h := sha1.New()
		h.Write([]byte(str))
		return fmt.Sprintf("%x", h.Sum(nil))
	}()

	if vali == signature {
		resp.Write([]byte(echostr))
	} else {
		resp.Write([]byte(""))
	}
	return
}

func WexinHandler(resp http.ResponseWriter, req *http.Request) {

	log.Println("method:", req.Method)
	if req.Method == "GET" {
		weixinValid(resp, req)
	} else {

		data, err := ioutil.ReadAll(req.Body)
		if nil != err {
			log.Println("read body err:", err)
			return
		}
		log.Println("data:", string(data))

		request := &entry.ReqMessage{}
		er := xml.Unmarshal(data, request)
		if nil != er {
			log.Println("decode body err:", er)
			return
		}

		event := request.Event
		msgType := request.MsgType
		ch := make(chan interface{})
		defer close(ch)
		if "event" == msgType && event == "subscribe" {
			go eventProcess(*request, ch)

		} else {
			var msg entry.TxtRequest
			err := xml.Unmarshal(data, &msg)
			if nil != err {
				log.Println("decode txt request body err:", er)
				return
			}

			go txtMessageProcess(msg, ch)
		}

		brespons, _ := xml.Marshal(<-ch)
		resp.Write(brespons)
	}
}

func eventProcess(msg entry.ReqMessage, ch chan interface{}) {
	resp := buildCoverPicMsg(msg)
	ch <- resp
}

func txtMessageProcess(msg entry.TxtRequest, ch chan interface{}) {
	code := msg.Content
	foods := query(10, code)
	var response interface{}
	if len(foods) <= 0 {
		resp := &entry.TxtResponse{}
		resp.FromUserName = msg.ToUserName
		resp.ToUserName = msg.FromUserName
		resp.MsgType = msg.MsgType
		resp.FuncFlag = 0
		resp.Content = "很遗憾你是吃货，没找到你的美食,你可以搜索爆米花!"
		resp.CreateTime = time.Duration(time.Now().Unix())
		response = resp

	} else {
		resp := &entry.PicResponse{}
		items := make([]*entry.Item, 0)

		for _, m := range foods {

			item := &entry.Item{}
			item.Title = m["name"].(string)
			item.PicUrl = m["img_url"].(string)
			item.Url = m["link"].(string)
			item.Description = m["name"].(string)
			items = append(items, item)
		}

		art := &entry.Articles{}
		art.Items = items
		resp.Articles = art
		resp.FromUserName = msg.ToUserName
		resp.ToUserName = msg.FromUserName
		resp.MsgType = "news"
		resp.FuncFlag = 1
		resp.CreateTime = time.Duration(time.Now().Unix())
		resp.ArticleCount = len(foods)
		response = resp
	}
	ch <- response
}

func query(limit int, code string) []mongo.M {
	conn, _ := pool.Get()
	db := &mongo.Database{conn, "meishi", mongo.DefaultLastErrorCmd}
	coll := db.C("foods")
	cursor, err := coll.Find(mongo.M{"name": mongo.M{"$regex": code}}).Limit(limit).Cursor()
	if nil != err {
		log.Println("query mongo fail |", err)
		return nil
	}

	defer cursor.Close()
	foods := make([]mongo.M, 0)
	i := 0

	for cursor.HasNext() && i < limit {
		var m mongo.M
		err := cursor.Next(&m)
		if nil != err {
			log.Println("decode mongo map fail|", err)
			continue
		}

		foods = append(foods, m)
		i++
	}
	return foods
}

func buildCoverPicMsg(msg entry.ReqMessage) entry.PicResponse {
	resp := entry.PicResponse{}
	items := make([]*entry.Item, 0)

	foods := query(1, "甜点")
	if len(foods) == 1 {
		m := foods[0]
		item := &entry.Item{}
		item.Title = m["name"].(string)
		item.PicUrl = m["img_url"].(string)
		item.Url = m["link"].(string)
		item.Description = welcome
		items = append(items, item)
	}

	art := &entry.Articles{}
	art.Items = items
	resp.Articles = art
	resp.FromUserName = msg.ToUserName
	resp.ToUserName = msg.FromUserName
	resp.MsgType = "news"
	resp.FuncFlag = 1
	resp.CreateTime = time.Duration(time.Now().Unix())
	resp.ArticleCount = len(foods)

	return resp
}
