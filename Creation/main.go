package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/asters1/tools"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type Ttsclient struct {
	lang   string
	name   string
	role   string
	rate   string
	volume string
	text   string
	format string
	user   string
}

func main() {
	var ttsc Ttsclient
	TTStokenUrl := "https://southeastasia.customvoice.api.speech.microsoft.com/api/texttospeech/v3.0-beta1/accdemopageentry/auth-token"
	TtsUrl := "https://southeastasia.customvoice.api.speech.microsoft.com/api/texttospeech/v3.0-beta1/accdemopage/speak"
	restoken := tools.RequestClient(TTStokenUrl, "get", "", "")
	ttstoken := gjson.Get(restoken, "authToken").String()
	//ttstoken := "6ade89b95df558ec056c4a49a738b9cf5ba0e6264b4fcb1805bcfe0d21bcdbb5"
	fmt.Println(ttstoken)
	r := gin.Default()
	r.POST("/tts", func(c *gin.Context) {
		ttsc.lang = c.PostForm("lang")
		ttsc.name = c.PostForm("name")
		ttsc.volume = c.PostForm("volume")
		ttsc.rate = c.PostForm("rate")
		ttsc.format = c.PostForm("format")
		ttsc.role = c.PostForm("role")
		ttsc.text = c.PostForm("text")
		ttsc.user = c.PostForm("user")
		ttsjson := strings.NewReader(`{
    "ssml": "<!--ID=B7267351-473F-409D-9765-754A8EBCDE05;Version=1|{\"VoiceNameToIdMapItems\":[{\"Id\":\"1011ca97-3e33-4e7c-8dda-a22dc244bafc\",\"Name\":\"Microsoft Server Speech Text to Speech Voice (` + ttsc.lang + `, ` + ttsc.name + `)\",\"ShortName\":\"` + ttsc.lang + `-` + ttsc.name + `\",\"Locale\":\"` + ttsc.lang + `\",\"VoiceType\":\"StandardVoice\"}]}-->\n<!--ID=5B95B1CC-2C7B-494F-B746-CF22A0E779B7;Version=1|{\"Locales\":{\"` + ttsc.lang + `\":{\"AutoApplyCustomLexiconFiles\":[{}]}}}-->\n<speak version=\"1.0\" xmlns=\"http://www.w3.org/2001/10/synthesis\" xmlns:mstts=\"http://www.w3.org/2001/mstts\" xmlns:emo=\"http://www.w3.org/2009/10/emotionml\" xml:lang=\"zh-CN\"><voice name=\"` + ttsc.lang + `-` + ttsc.name + `\"><lang xml:lang=\"` + ttsc.lang + `\"><mstts:express-as style=\"\" styledegree=\"1.0\" role=\"` + ttsc.role + `\"><prosody rate=\"` + ttsc.rate + `%\" volume=\"` + ttsc.volume + `%\">` + ttsc.text + `</prosody></mstts:express-as></lang></voice></speak>",
    "ttsAudioFormat": "` + ttsc.format + `",
    "offsetInPlainText": 0,
    "lengthInPlainText":300,"properties": {
        "SpeakTriggerSource": "AccTuningPagePlayButton"
    }
}`)
		req, _ := http.NewRequest("POST", TtsUrl, ttsjson)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accdemopageauthtoken", ttstoken)

		response, _ := http.DefaultClient.Do(req)
		body, _ := ioutil.ReadAll(response.Body)
		ioutil.WriteFile("./mp3/"+ttsc.user+".mp3", body, 0666)
		response.Body.Close()
		c.Header("Content-Type", "audio/mpeg")
		c.File("./mp3/" + ttsc.user + ".mp3")
		//c.Redirect(http.StatusMovedPermanently, "http://localhost:8080/mp3?user="+ttsc.user)
	})
	r.Run()

}