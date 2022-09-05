package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asters1/tools"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	uuid string
	conn *websocket.Conn
}

var (
	lock    sync.Mutex
	wg      sync.WaitGroup
	c2      Client
	ch      chan string
	check   chan int
	oldTime int64
	nowTime int64
)

func GetISOTime() string {
	T := time.Now().String()
	return T[:23][:10] + "T" + T[:23][11:] + "Z"

}
func GetLogTime() string {
	BJ, _ := time.LoadLocation("Asia/Shanghai")
	return "[" + time.Now().In(BJ).String()[:19] + "]"
}
func NewClient() (*Client, error) {
	//	token := GetToken()
	uuid := tools.GetUUID()
	//	WssUrl := `wss://all.wisteria.cf/eastus.tts.speech.microsoft.com/cognitiveservices/websocket/v1?Authorization=` + token + `&X-ConnectionId=` + uuid
	//	wss://eastus.api.speech.microsoft.com/cognitiveservices/websocket/v1?TricType=AzureDemo&Authorization=bearer undefined&X-ConnectionId=8188D30D0FCA4
	WssUrl := `wss://eastus.api.speech.microsoft.com/cognitiveservices/websocket/v1?TricType=AzureDemo&Authorization=bearer%20undefined&X-ConnectionId=` + uuid

	dl := websocket.Dialer{
		EnableCompression: true,
	}
	conn, _, err := dl.Dial(WssUrl, tools.GetHeader(
		`Accept-Encoding:gzip
		User-Agent:Mozilla/5.0 (Linux; Android 7.1.2; M2012K11AC Build/N6F26Q; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/81.0.4044.117 Mobile Safari/537.36
		Origin:https://azure.microsoft.com`,
	))
	if err != nil {
		return nil, err
	}
	m1 := "Path: speech.config\r\nX-RequestId: " + uuid + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/json\r\n\r\n{\"context\":{\"system\":{\"name\":\"SpeechSDK\",\"version\":\"1.19.0\",\"build\":\"JavaScript\",\"lang\":\"JavaScript\",\"os\":{\"platform\":\"Browser/Linux x86_64\",\"name\":\"Mozilla/5.0 (X11; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0\",\"version\":\"5.0 (X11)\"}}}}"
	m2 := "Path: synthesis.context\r\nX-RequestId: " + uuid + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/json\r\n\r\n{\"synthesis\":{\"audio\":{\"metadataOptions\":{\"sentenceBoundaryEnabled\":false,\"wordBoundaryEnabled\":false},\"outputFormat\":\"audio-24khz-160kbitrate-mono-mp3\"}}}"
	lock.Lock()
	conn.WriteMessage(websocket.TextMessage, []byte(m1))
	conn.WriteMessage(websocket.TextMessage, []byte(m2))
	lock.Unlock()
	oldTime = time.Now().Unix()

	return &Client{
		uuid: uuid,
		conn: conn,
	}, err
}
func (c Client) Close() {
	c.conn.Close()
}
func RestConn(c *Client) {
	lock.Lock()
	c.conn.Close()
	for {
		c1, err1 := NewClient()
		if err1 == nil {

			*c = *c1
			fmt.Println(GetLogTime() + " -> 连接已重置...")
			break
		}
	}
	lock.Unlock()
}
func CheckConn(c *Client) {
	for {

		lock.Lock()
		err := c.conn.WriteMessage(websocket.PingMessage, []byte(""))
		lock.Unlock()
		if err != nil {
			fmt.Println(GetLogTime() + " -> 连接断开...")

			RestConn(c)
		} else {
			fmt.Println(GetLogTime() + " -> 连接正常...")
		}
		time.Sleep(time.Second * 10)

	}
}
func SendEmptyMessage(c *Client) {
	for {
		nowTime = time.Now().Unix()
		ti := nowTime - oldTime
		fmt.Println(ti)
		if nowTime-oldTime > 30 {
			SSML := `<speak xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="http://www.w3.org/2001/mstts" xmlns:emo="http://www.w3.org/2009/10/emotionml" version="1.0" xml:lang="en-US">
			<voice name="zh-CN-YunXiNeural">
			<mstts:express-as style="general" >
			<prosody rate="0%" volume="100" pitch="0%"></prosody>
			</mstts:express-as>
			</voice>
			</speak>`
			m3 := "Path: ssml\r\nX-RequestId: " + tools.GetUUID() + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/ssml+xml\r\n\r\n" + SSML
			lock.Lock()
			c.conn.WriteMessage(websocket.TextMessage, []byte(m3))
			lock.Unlock()
		}
		time.Sleep(time.Second * 30)
	}

}
func RunWebSocket() {

	fmt.Println("创建websocket连接...")
	Client, err := NewClient()
	if err != nil {

		fmt.Println("创建websocket连接失败...")
		return
	} else {
		fmt.Println("创建websocket连接成功!")
	}
	defer Client.Close()
	go CheckConn(Client)
	go SendEmptyMessage(Client)

	for {
		language := <-ch
		name := <-ch
		volume := <-ch
		rate := <-ch

		pitch := <-ch
		text := <-ch
		sjc := <-ch

		SSML := `<speak xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="http://www.w3.org/2001/mstts" xmlns:emo="http://www.w3.org/2009/10/emotionml" version="1.0" xml:lang="en-US">
		<voice name="` + language + `-` + name + `">
		<mstts:express-as style="general" >
		<prosody rate="` + rate + `%" volume="` + volume + `" pitch="` + pitch + `%">` + string(text) + `</prosody>
		</mstts:express-as>
		</voice>
		</speak>`
		m3 := "Path: ssml\r\nX-RequestId: " + tools.GetUUID() + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/ssml+xml\r\n\r\n" + SSML
		lock.Lock()
		Client.conn.WriteMessage(websocket.TextMessage, []byte(m3))
		lock.Unlock()
		oldTime = time.Now().Unix()

		var Adata []byte
		fmt.Println("正在下载文件...")
		for {
			Num, message, err := Client.conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				break
			}
			if Num == 2 {
				index := strings.Index(string(message), "Path:audio")

				data := []byte(string(message)[index+12:])
				Adata = append(Adata, data...)
			} else if Num == 1 && string(message)[len(string(message))-14:len(string(message))-6] == "turn.end" {
				fmt.Println("已完成")
				break
			}

		}
		if len(Adata) > 2400 {
			Adata = Adata[:len(Adata)-2400]
			ioutil.WriteFile("./mp3/"+sjc+".mp3", Adata, 0666)
			check <- 0
		} else {
			check <- 1
			RestConn(Client)

		}
	}
}

func main() {
	ch = make(chan string)
	check = make(chan int)
	go RunWebSocket()
	r := gin.Default()
	r.POST("/tts", func(c *gin.Context) {
		ch <- c.PostForm("language")
		ch <- c.PostForm("name")
		ch <- c.PostForm("volume")
		ch <- c.PostForm("rate")

		ch <- c.PostForm("pitch")
		ch <- c.PostForm("text")
		sjc := c.PostForm("sjc")
		ch <- sjc
		a := <-check
		if a == 0 {
			fmt.Println("音频转换已完成")
			c.Redirect(http.StatusMovedPermanently, "http://localhost:8080/mp3?sjc="+sjc)
		} else if a == 1 {
			fmt.Println("音频转换失败!")

		}

	})
	r.GET("/mp3", func(ctx *gin.Context) {
		sjc := ctx.Query("sjc")
		ctx.Header("Content-Type", "audio/mpeg")
		ctx.File("./mp3/" + sjc + ".mp3")

	})
	r.Run()

}
