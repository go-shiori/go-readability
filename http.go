// 改自 https://github.com/kingwkb/readability python版本
// 于2016-11-10，睡不着了，写起代码来就到零晨4点了
// by: ying32 E-mail:1444386932@qq.com
package readability

import (
	"compress/flate"
	"compress/gzip"

	"io/ioutil"
	"net/http"
	"strings"

	"github.com/axgle/mahonia"
)

func httpGet(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	contentEncoding := strings.Trim(strings.ToLower(resp.Header.Get("Content-Encoding")), " ")
	var bytes []byte
	if contentEncoding == "gzip" {
		x, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		bytes, err = ioutil.ReadAll(x)
		if err != nil {
			return "", err
		}
	} else if contentEncoding == "deflate" {
		x := flate.NewReader(resp.Body)
		bytes, err = ioutil.ReadAll(x)
		if err != nil {
			return "", err
		}
	} else {
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
	}

	srcStr := string(bytes)
	pageCodes := pageCodeReg.FindStringSubmatch(srcStr)
	if len(pageCodes) >= 2 {
		curCode := strings.ToLower(pageCodes[1])
		if curCode == "gb2312" || curCode == "gbk" {
			decoder := mahonia.NewDecoder("gbk")
			srcStr = decoder.ConvertString(srcStr)
		}
	}
	return srcStr, nil
}
