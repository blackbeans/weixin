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
	ToUserName   string // base struct
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

type LocRequest struct {
	ReqMessage
	Location_X float64
	Location_Y float64
	Scale      int32
	Label      string
}

type TxtResponse struct {
	XMLName xml.Name `xml:"xml"`
	RespMessage
	Content string
}

type PicResponse struct {
	XMLName xml.Name `xml:"xml"`
	RespMessage
	ArticleCount int
	Articles     *Articles
}

type Item struct {
	Title       string
	Description string
	PicUrl      string
	Url         string
}

type Articles struct {
	// Articles xml.Name `xml:"Articles"`
	Items []*Item `xml:"item"`
}
