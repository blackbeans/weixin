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
	welcome    = "欢迎使用美食助手应用，您可以属于希望的文字获得美食、发送地理位置可以查询附近餐馆哦！愿你在这里发现生活真正的意义~业务合作微信账号:blackbeans"
)

var pool *mongo.Pool

func init() {
	pool = mongo.NewDialPool("42.96.167.9:27018", 1000)
}

// func main() {

// 	msg := entry.LocRequest{}
// 	msg.Location_X = 23.312
// 	msg.Location_Y = 120.8
// 	locMessageProcess(msg, nil)
// }

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
			//添加关注事件
			go subEventProcess(*request, ch)

		} else if "event" == msgType && event == "unsubscribe" {
			//取消订阅
			go unsubEventProcess(*request, ch)

		} else if "location" == msgType {
			//地理位置
			var msg entry.LocRequest
			err := xml.Unmarshal(data, msg)
			if nil != err {
				log.Println("decode txt request body err:", er)
				return
			}

			go locMessageProcess(msg, ch)

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
		log.Println(string(brespons))
		resp.Write(brespons)
	}
}

func locMessageProcess(msg entry.LocRequest, ch chan interface{}) {
	conn, _ := pool.Get()
	db := &mongo.Database{conn, "search", mongo.DefaultLastErrorCmd}
	coll := db.C("resturant")
	cond := mongo.M{"$near": mongo.A{msg.Location_X, msg.Location_Y}}
	fields := mongo.M{"name": 1, "province": 1, "city": 1, "district": 1, "_id": 0, "description.tel": 1}
	cursor, err := coll.Find(mongo.M{"geoloc": cond}).Fields(fields).Limit(1).Cursor()
	if nil != err {
		log.Println("query mongo fail |", err)
		return
	}

	defer cursor.Close()

	locs := traverseQueryResult(cursor, 10)
	resp := buildCoverPicMsg(msg.ReqMessage)
	for _, val := range locs {
		shop := resp.Articles.Items[0]
		shop.Title = fmt.Sprintf("离你最近的餐馆 ：%x(电话:%x)", val["name"].(string), val["tel"].(string))
		shop.Description = fmt.Sprintf("地址:%s,%s,%s", val["province"], val["city"], val["district"])
	}
	ch <- resp

}

func subEventProcess(msg entry.ReqMessage, ch chan interface{}) {
	resp := buildCoverPicMsg(msg)
	ch <- resp
}

func unsubEventProcess(msg entry.ReqMessage, ch chan interface{}) {
	resp := buildTxtMsg("感谢您的关注，希望您下次继续光顾本应用^_^!", "text")
	resp.FromUserName = msg.ToUserName
	resp.ToUserName = msg.FromUserName
	ch <- *resp
}

func txtMessageProcess(msg entry.TxtRequest, ch chan interface{}) {
	code := msg.Content
	foods := query(10, code)
	var response interface{}
	if len(foods) <= 0 {
		resp := buildTxtMsg("很遗憾你是吃货，没找到你的美食,你可以搜索爆米花!", msg.MsgType)
		resp.FromUserName = msg.ToUserName
		resp.ToUserName = msg.FromUserName
		response = *resp
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

	return traverseQueryResult(cursor, limit)
}

func traverseQueryResult(cursor mongo.Cursor, limit int) []mongo.M {
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

func buildTxtMsg(content string, msgType string) *entry.TxtResponse {
	resp := &entry.TxtResponse{}
	resp.MsgType = msgType
	resp.FuncFlag = 0
	resp.Content = content
	resp.CreateTime = time.Duration(time.Now().Unix())
	return resp
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
