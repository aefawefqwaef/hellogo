package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Function to fetch HTML content from a URL
func fetchHTML(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

type News struct {
	Img  string `json:"img"`
	Text string `json:"text"`
}

var NewsList []News

// GetImageBase64 从给定的 URL 获取图片并将其转换为 Base64 编码字符串
func GetImageBase64(url string) string {
	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the image:", err)
		return ""
	}
	defer resp.Body.Close()

	// 检查 HTTP 响应状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Non-OK HTTP status:", resp.StatusCode)
		return ""
	}

	// 读取响应体
	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the image data:", err)
		return ""
	}

	// 将图片数据编码为 Base64 字符串
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	return base64Image
}

// Function to extract content and image URL
func extractContentAndImageURL(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "tgme_widget_message_photo_wrap") {
				for _, a := range n.Attr {
					if a.Key == "style" && strings.Contains(a.Val, "background-image") {
						start := strings.Index(a.Val, "url('") + 5
						end := strings.Index(a.Val, "')")
						imageURL := a.Val[start:end]
						fmt.Printf("Background Image URL: %s\n", imageURL)
						imgData := GetImageBase64(imageURL)
						NewsList = append(NewsList, News{Img: imgData})
					}
				}
			}
		}
	}

	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "tgme_widget_message_text") && !strings.Contains(attr.Val, "message_reply_text") {
				var contentBuilder strings.Builder
				extractText(n, &contentBuilder)
				content := contentBuilder.String()
				content = strings.ReplaceAll(content, "<br />", "\n")
				fmt.Printf("Content: %s\n", content)
				fmt.Printf("==============\n\n")
				if len(NewsList) == 0 || NewsList[len(NewsList)-1].Text != "" {
					NewsList = append(NewsList, News{})
				}
				NewsList[len(NewsList)-1].Text = base64.StdEncoding.EncodeToString([]byte(content))
				// NewsList[len(NewsList)-1].Text = content
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractContentAndImageURL(c)
	}
}

// Function to extract text content
func extractText(n *html.Node, builder *strings.Builder) {
	imgRefLevel := false
	if n.Type == html.TextNode {
		builder.WriteString(n.Data)
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				builder.WriteString(fmt.Sprintf("[%s](%s)", n.FirstChild.Data, attr.Val))
				imgRefLevel = true
			}
		}
	}
	if imgRefLevel {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, builder)
	}
}

func fetchNews(url string) {
	doc, err := fetchHTML(url)
	if err != nil {
		fmt.Printf("Error fetching HTML: %v\n", err)
	}

	extractContentAndImageURL(doc)
}

type Config struct {
	Version [3]int   `json:"version"`
	Hash    []string `json:"hash"`
}

func ReadJSONFile(filename string, v any) error {
	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取文件内容
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// 解析JSON数据
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	return nil
}

func SaveToJSONFile(filename string, data interface{}) error {
	// 将数据编码为 JSON 格式
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 将 JSON 数据写入文件
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}
func CalculateMD5(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

func main() {
	for {

		NewsList = []News{}
		fetchNews("https://t.me/s/tnews365")

		var config Config
		ReadJSONFile("config.json", &config)
		md5s := make(map[string]bool)
		for _, md5 := range config.Hash {
			md5s[md5] = true
		}
		var unreadNews []News
		for _, news := range NewsList {
			md5 := CalculateMD5(news.Img + news.Text)
			if exist := md5s[md5]; !exist {
				unreadNews = append(unreadNews, news)
			}
		}
		if len(unreadNews) == 0 {
			continue
		}

		for _, news := range unreadNews {
			md5 := CalculateMD5(news.Img + news.Text)
			config.Hash = append(config.Hash, md5)
		}

		// 将 NewsList 转换为 JSON
		jsonData, err := json.MarshalIndent(NewsList, "", "  ")
		if err != nil {
			fmt.Println("Error marshalling to JSON:", err)
			return
		}

		// 将 JSON 数据写入文件
		file, err := os.Create("news.json")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		_, err = file.Write(jsonData)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		fmt.Println("JSON data successfully written to news.json")

		output, err := exec.Command("git", "commit", "-am", "update").CombinedOutput()
		log.Printf("cmd %v %v", err, output)
		output, err = exec.Command("git", "push").CombinedOutput()
		log.Printf("cmd %v %v", err, output)
		output, err = exec.Command("git", "tag", fmt.Sprintf("v%v.%v.%v", config.Version[0], config.Version[1], config.Version[2]+1)).CombinedOutput()
		log.Printf("cmd %v %v", err, output)
		output, err = exec.Command("git", "push", "origin", "tag", fmt.Sprintf("v%v.%v.%v", config.Version[0], config.Version[1], config.Version[2]+1)).CombinedOutput()
		log.Printf("cmd %v %v", err, output)
		if err != nil {

			continue
		}
		config.Version[2] += 1
		SaveToJSONFile("config.json", config)

		time.Sleep(time.Minute * 10)
	}
}
