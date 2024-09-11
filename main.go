package hellogo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

func main() {
	for {

		NewsList = []News{}
		fetchNews("https://t.me/s/tnews365")

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
		time.Sleep(time.Minute * 10)
	}
}
