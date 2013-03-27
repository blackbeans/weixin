package entry

import (
	"encoding/xml"
	"time"
)

type ReqMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   // base struct
	FromUserName string
	MsgType      string
	CreateTime   time.Duration
	Event        string
	EventKey     string
}

type RespMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   // base struct
	FromUserName string
	MsgType      string
	CreateTime   time.Duration
	FuncFlag     int
}

type TxtRequest struct {
	ReqMessage
	Content string
	MsgId   int
}

type TxtResponse struct {
	RespMessage
	Content string
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
	RespMessage
	ArticleCount int
	Articles     *Articles
}
