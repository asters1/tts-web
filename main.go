package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/asters1/tools"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
func NewConn() *websocket.Conn {

	fmt.Println("获取token...")
	token := GetToken()
	uuid := tools.GetUUID()
	WssUrl := `wss://all.wisteria.cf/eastus.tts.speech.microsoft.com/cognitiveservices/websocket/v1?Authorization=` + token + `&X-ConnectionId=` + uuid
	dl := websocket.Dialer{
		EnableCompression: true,
	}

	fmt.Println("创建websocket连接...")
	conn, _, err := dl.Dial(WssUrl, tools.GetHeader(
		`Accept-Encoding:gzip
		User-Agent:Mozilla/5.0 (Linux; Android 7.1.2; M2012K11AC Build/N6F26Q; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/81.0.4044.117 Mobile Safari/537.36
		Origin:https://azure.microsoft.com`,
	))
	if err != nil {
		fmt.Println("websocket连接失败")
	}

	m1 := "Path: speech.config\r\nX-RequestId: " + uuid + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/json\r\n\r\n{\"context\":{\"system\":{\"name\":\"SpeechSDK\",\"version\":\"1.19.0\",\"build\":\"JavaScript\",\"lang\":\"JavaScript\",\"os\":{\"platform\":\"Browser/Linux x86_64\",\"name\":\"Mozilla/5.0 (X11; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0\",\"version\":\"5.0 (X11)\"}}}}"
	m2 := "Path: synthesis.context\r\nX-RequestId: " + uuid + "\r\nX-Timestamp: " + GetISOTime() + "\r\nContent-Type: application/json\r\n\r\n{\"synthesis\":{\"audio\":{\"metadataOptions\":{\"sentenceBoundaryEnabled\":false,\"wordBoundaryEnabled\":false},\"outputFormat\":\"audio-24khz-160kbitrate-mono-mp3\"}}}"
	conn.WriteMessage(websocket.TextMessage, []byte(m1))
	conn.WriteMessage(websocket.TextMessage, []byte(m2))
	return conn
}
func main() {
	ch := make(chan string)
	check := make(chan int)
	go func() {
		conn := NewConn()
		defer conn.Close()
		go func() {
			for {
				err := conn.WriteMessage(websocket.PingMessage, []byte(""))
				if err != nil {
					fmt.Println("连接断开正在重连")
					conn1 := NewConn()
					*conn = *conn1
				} else {
					fmt.Println("连接正常")
				}
				time.Sleep(time.Second * 20)
			}
		}()
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
			conn.WriteMessage(websocket.TextMessage, []byte(m3))

			var Adata []byte
			fmt.Println("正在下载文件...")
			for {
				Num, message, err := conn.ReadMessage()
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
			Adata = Adata[:len(Adata)-2400]
			ioutil.WriteFile("./mp3/"+sjc+".mp3", Adata, 0666)
			check <- 0

		}
	}()
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
	r.GET("mp3", func(ctx *gin.Context) {
		sjc := ctx.Query("sjc")
		ctx.Header("Content-Type", "audio/mpeg")
		ctx.File("./mp3/" + sjc + ".mp3")

	})
	r.Run()
}
