package Spy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type Spy struct {
	filter   func(*Response)
	savedata func(*Response)
}

type Response struct {
	backNode   *Response
	spy        *Spy
	resp       *http.Response
	Body       string
	Cookies    map[string]string
	Data       map[string]string
	option     *Option
	StatusCode int
	nextNode   *Response
}

func (w *Response) GetMethod() string {
	return w.option.Method
}

func (w *Response) FindAllString(regstr string) []string {
	reg := regexp.MustCompile(regstr).FindAllString(w.Body, -1)
	return reg
}

func (w *Response) FindAllStringSubmatch(regstr string) [][]string {
	reg := regexp.MustCompile(regstr).FindAllStringSubmatch(w.Body, -1)
	if len(reg) == 0 {
		return [][]string{}
	}
	return reg
}

func (w *Response) GetHeader(key string) string {
	for w.resp == nil {
	}
	return w.resp.Header.Get(key)
}

type Option struct {
	Url    string
	Method string
	Data   map[string][]string
	Header map[string]string
	Cookie string
}

var headNode *Response = nil

func (spy *Spy) Filter(fun func(*Response)) {
	spy.filter = fun
}

func (spy *Spy) SaveData(fun func(*Response)) {
	spy.savedata = fun
}

func NewSpy() *Spy {
	var newSpy Spy
	return &newSpy
}

func Add(spy *Spy, url string, option *Option) *Response {
	rs := &Response{}
	rs.spy = spy
	rs.option = option
	rs.option.Url = url
	rs.Cookies = map[string]string{}
	rs.Data = map[string]string{}
	if headNode == nil {
		headNode = rs
		headNode.backNode = nil
		headNode.nextNode = nil
	} else {
		if headNode.nextNode == nil {
			headNode.nextNode = rs
			rs.backNode = headNode
		} else {
			newestNode := headNode.backNode
			newestNode.nextNode = rs
			rs.backNode = newestNode
		}
		rs.nextNode = nil
		headNode.backNode = rs
	}
	return rs
}

func downLoader(rs *Response) error {
	if rs.option.Url == "" {
		return errors.New("Fail,Url is empty.")
	}
	if rs.option.Method == "" {
		rs.option.Method = "Get"
	}
	var method string
	var resp *http.Response
	var err error
	switch strings.ToLower(rs.option.Method) {
	case "get":
		if resp, err = http.Get(rs.option.Url); err != nil {
			return err
		}
	case "post":
		if resp, err = http.PostForm(rs.option.Url, rs.option.Data); err != nil {
			return err
		}
	default:
		return errors.New("Url Method Type Fail.")
	}
	resp.Header.Set("Cookie", rs.option.Cookie)
	fmt.Println(method)
	for k, v := range rs.option.Header {
		resp.Header.Set(k, v)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	rs.Body = string(body)
	rs.StatusCode = resp.StatusCode

	for _, v := range resp.Cookies() {
		rs.Cookies[v.Name] = v.Value
	}
	rs.resp = resp
	return nil
}

func getNode() *Response {
	node := headNode
	if node != nil {
		if node.nextNode == nil {
			headNode = nil
			return node
		}
		headNode = node.nextNode
		if headNode.nextNode == nil {
			headNode.backNode = nil
		} else {
			headNode.backNode = node.backNode
		}
	} else {
		return nil
	}
	return node
}

func runNode(node *Response) {
	err := downLoader(node)
	if err != nil {
		fmt.Println(node.option.Url, "Open Error.", err)
		waitgroup.Done()
		return
	}
	if node.spy.filter != nil {
		node.spy.filter(node)
	}
	if node.spy.savedata != nil {
		node.spy.savedata(node)
	}
	waitgroup.Done()
	newNode := getNode()
	if newNode != nil {
		waitgroup.Add(1)
		go runNode(newNode)
	}
}

var waitgroup sync.WaitGroup

func Run(process int) {
	for i := 0; i < process; i++ {
		node := getNode()
		if node != nil {
			waitgroup.Add(1)
			go runNode(node)
		} else {
			break
		}
	}
	waitgroup.Wait()
	fmt.Println("Success!")
}
