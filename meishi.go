package main

import (
	"crypto/sha1"
	"encoding/xml"
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

type Request struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   // base struct
	FromUserName string
	CreateTime   time.Duration
	MsgType      string
	Content      string
	MsgId        int
}

type TxtResponse struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   // base struct
	FromUserName string
	Content      string
	CreateTime   time.Duration
	MsgType      string
	FuncFlag     int
}

type Item struct {
	Title       string
	Description string
	PicUrl      string
	Url         string
}

type Articles struct {
	Articles xml.Name `xml:"Articles"`
	Items    []*Item  `xml:"item"`
}

type PicResponse struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string
	FromUserName string
	CreateTime   time.Duration
	MsgType      string
	ArticleCount int
	Articles     *Articles

	FuncFlag int
}

const (
	basiurl    = "http://redis.io/commands/"
	forwardurl = "http://localhost:8080/"
	token      = "betago"
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

		request := &Request{}
		er := xml.Unmarshal(data, request)
		if nil != er {
			log.Println("decode body err:", er)
			return
		}

		code := request.Content

		conn, _ := pool.Get()
		db := &mongo.Database{conn, "meishi", mongo.DefaultLastErrorCmd}
		coll := db.C("foods")
		cursor, err := coll.Find(mongo.M{"name": mongo.M{"$regex": code}}).Limit(10).Cursor()
		if nil != err {
			log.Println("query mongo fail |", err)
			return
		}
		defer cursor.Close()
		foods := make([]mongo.M, 0)
		i := 0

		for cursor.HasNext() && i < 10 {
			var m mongo.M
			err := cursor.Next(&m)
			if nil != err {
				log.Println("decode mongo map fail|", err)
				continue
			}

			foods = append(foods, m)
			// log.Println(m)
			i++
		}

		if i <= 0 {
			response := &TxtResponse{}
			response.FromUserName = request.ToUserName
			response.ToUserName = request.FromUserName
			response.MsgType = request.MsgType
			response.FuncFlag = 0
			response.Content = "很遗憾你是吃货，没找到你的美食,你可以搜索爆米花!"
			response.CreateTime = time.Duration(time.Now().Unix())

			write(resp, response)
		} else {
			response := &PicResponse{}
			items := make([]*Item, 0)

			for _, m := range foods {

				log.Println("food:", m)
				item := &Item{}
				item.Title = m["name"].(string)
				item.PicUrl = m["img_url"].(string)
				item.Url = m["link"].(string)
				item.Description = m["name"].(string)
				items = append(items, item)
			}

			art := &Articles{}
			art.Items = items
			response.Articles = art
			response.FromUserName = request.ToUserName
			response.ToUserName = request.FromUserName
			response.MsgType = "news"
			response.FuncFlag = 1
			response.CreateTime = time.Duration(time.Now().Unix())
			response.ArticleCount = len(foods)
			write(resp, response)
		}
	}
}

func write(resp http.ResponseWriter, obj interface{}) {
	brespons, _ := xml.Marshal(obj)
	log.Println(string(brespons))
	resp.Write(brespons)
}
