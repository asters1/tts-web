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
	lock  sync.Mutex
	wg    sync.WaitGroup
	c2    Client
	ch    chan string
	check chan int
)

func GetToken() string {
	res := tools.RequestClient("https://all.wisteria.cf/azure.microsoft.com/en-gb/services/cognitive-services/text-to-speech/", "get", "", "")
	token := tools.Re(res, `token: \"(.*?)\"`)[1]
	return token

}
func GetISOTime() string {
	T := time.Now().String()
	return T[:23][:10] + "T" + T[:23][11:] + "Z"

}
func GetLogTime() string {
	BJ, _ := time.LoadLocation("Asia/Shanghai")
	return "[" + time.Now().In(BJ).String()[:19] + "]"
}
func NewClient() (*Client, error) {
	token := GetToken()
	uuid := tools.GetUUID()
	WssUrl := `wss://all.wisteria.cf/eastus.tts.speech.microsoft.com/cognitiveservices/websocket/v1?Authorization=` + token + `&X-ConnectionId=` + uuid

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
	conn.WriteMessage(websocket.TextMessage, []byte(m1))
	conn.WriteMessage(websocket.TextMessage, []byte(m2))

	return &Client{
		uuid: uuid,
		conn: conn,
	}, err
}
func (c Client) Close() {
	c.conn.Close()
}
func RestConn(c *Client) {
	c.conn.Close()
	for {
		c1, err1 := NewClient()
		if err1 == nil {

			*c = *c1
			fmt.Println(GetLogTime() + " -> 连接已重置...")
			break
		}
	}
}
func CheckConn(c *Client) {
	for {
		err := c.conn.WriteMessage(websocket.PingMessage, []byte(""))
		if err != nil {
			fmt.Println(GetLogTime() + " -> 连接断开...")
			time.Sleep(time.Second * 5)

			RestConn(c)
		} else {
			fmt.Println(GetLogTime() + " -> 连接正常...")
		}
		time.Sleep(time.Second * 5)
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
		Client.conn.WriteMessage(websocket.TextMessage, []byte(m3))

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
		}

		c.Redirect(http.StatusMovedPermanently, "http://localhost:8080/mp3?sjc="+sjc)

	})
	r.GET("/mp3", func(ctx *gin.Context) {
		sjc := ctx.Query("sjc")
		ctx.Header("Content-Type", "audio/mpeg")
		ctx.File("./mp3/" + sjc + ".mp3")

	})
	r.Run()

}
