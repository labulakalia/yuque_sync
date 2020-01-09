package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	yq yuquesync
)

const (
	hugocfg   = "config.toml"
	mainpath  = "content"
	imagepath = "images"
	article   = "post"
	hugocmd   = "hugo"
)

type Docs struct {
	Data []data `json:"data"`
}

type data struct {
	Id          int    `json:"id"`
	Body        string `json:"body,omitempty"`
	PublishedAt string `json:"published_at"`
	Title       string `json:"title"`
}

type DocsContent struct {
	Data data `json:"data"`
}

func ReqGet(uri string) ([]byte, error) {
	url := yq.YuQue.Api + uri
	log.Println("Start Req Url: ", url)
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("User-Agent", "blog")
	if yq.YuQue.Token != "" {
		r.Header.Add("X-Auth-Token", yq.YuQue.Token)
	}
	client := http.Client{Timeout: 60 * time.Second}

	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Err Status Code: %d", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// 获取所有的文章ID
func getAllDocs(user string, path string) (*Docs, error) {
	alldocsid := Docs{}
	uri := fmt.Sprintf("/repos/%s/%s/docs/", user, path)

	body, err := ReqGet(uri)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &alldocsid)
	if err != nil {
		return nil, err
	}
	return &alldocsid, nil
}

func getDocsDetail(user, kb string, id int) (*DocsContent, error) {
	if user == "" || kb == "" {
		return nil, errors.New("Please input user and docs path")
	}
	uri := fmt.Sprintf("/repos/%s/%s/docs/%d?raw=1", user, kb, id)
	docscontent := DocsContent{}

	body, err := ReqGet(uri)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &docscontent)
	if err != nil {
		return nil, err
	}
	return &docscontent, nil
}

func downloadDocs(storepath string, docs *DocsContent) error {
	err := replaceimageurl(docs)
	if err != nil {
		log.Printf("replaceimageurl failed: %+v\n", err)
		return err
	}
	var buf bytes.Buffer

	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", docs.Data.Title))
	buf.WriteString(fmt.Sprintf("date: %s\n", docs.Data.PublishedAt))
	buf.WriteString("draft: false\n")
	buf.WriteString("---\n")
	buf.WriteString("\n")
	buf.WriteString("\n")

	buf.WriteString(docs.Data.Body)

	file, err := os.Create(storepath)
	if err != nil {
		return err
	}

	_, err = file.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return file.Sync()
}

// 替换语雀的照片
func replaceimageurl(docs *DocsContent) error {
	replregex, err := regexp.Compile(`\((https://cdn\.nlark\.com/yuque.*?\))`)
	if err != nil {
		return err
	}
	imagenumber := 1
	for {
		url := replregex.FindAllString(docs.Data.Body, 1)
		if len(url) == 0 || len(url) != 1 {
			break
		}

		imageallpath := fmt.Sprintf("%s/%s/%d_%d.png", mainpath, imagepath, docs.Data.Id, imagenumber)
		log.Printf("Start Replace %s image\n", imageallpath)
		err := downimage(imageallpath, url[0])
		if err != nil {
			log.Printf("downimage failed url: %s err: %+v\n", url[0], err)
			continue
		}
		docs.Data.Body = strings.Replace(docs.Data.Body, url[0], fmt.Sprintf("(/%s/%d_%d.png)", imagepath, docs.Data.Id, imagenumber), 1)
		imagenumber++
	}
	log.Println("Replace All Image success")
	return nil
}

// down picture to disk
func downimage(picpath, url string) error {
	url = strings.TrimLeft(url, "(")
	url = strings.TrimRight(url, ")")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	file, err := os.Create(picpath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func mkdirpath(dir string) error {

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("MkdirAll dir %s failed: %+v\n", dir, err)
			return err
		}
	}
	return nil
}

func httpwebhook() {
	http.HandleFunc("/yuque", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("ReadAll body: %+v\n", err)
			w.WriteHeader(500)
			return
		}
		docscontent := DocsContent{}
		err = json.Unmarshal(body, &docscontent)
		if err != nil {
			log.Printf("Unmarshal body: %+v\n", err)
			w.WriteHeader(500)
			return
		}
		downfilepath := fmt.Sprintf("%s/%s/%d.md", mainpath, article, docscontent.Data.Id)
		err = downloadDocs(downfilepath, &docscontent)
		if err != nil {
			log.Printf("Download Title %s Err: %v", docscontent.Data.Title, err)
			w.WriteHeader(500)
			return
		}
		runhugocmd()
	})
	addr := fmt.Sprintf(":%d", yq.YuQue.Port)
	log.Printf("Start Webhooks API %s/yuque", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func runhugocmd() {
	cmds := []string{hugocmd}
	if yq.YuQue.AfterCmd != "" {
		cmds = append(cmds, yq.YuQue.AfterCmd)
	}
	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		cmd := exec.Command("bash", "-c", cmd)
		err := cmd.Run()
		if err != nil {
			log.Printf("run hugo command failed: %+v\n", err)
		}
	}

}

type yuque struct {
	User     string `toml:"user"`
	Kb       string `toml:"kb"`
	Token    string `toml:"token"`
	Api      string `toml:"api"`
	Port     int    `toml:"port"`
	AfterCmd string `toml:"aftercmd"`
}

type yuquesync struct {
	YuQue yuque `toml:"yuque-sync"`
}

func main() {
	_, err := toml.DecodeFile(hugocfg, &yq)
	if err != nil {
		log.Fatal(err)
	}

	if yq.YuQue.User == "" || yq.YuQue.Kb == "" {
		log.Fatalln("Please input user and docs path")
	}
	err = mkdirpath(fmt.Sprintf("%s/%s", mainpath, imagepath))
	if err != nil {
		log.Fatalln(err)
	}
	err = mkdirpath(fmt.Sprintf("%s/%s", mainpath, article))
	if err != nil {
		log.Fatalln(err)
	}

	docs, err := getAllDocs(yq.YuQue.User, yq.YuQue.Kb)
	if err != nil {
		log.Fatal("Get All Docs Id Failed: ", err)
	}
	log.Println("Total Get Article", len(docs.Data))

	for _, docs := range docs.Data {
		time.Sleep(time.Millisecond * 300)
		res, err := getDocsDetail(yq.YuQue.User, yq.YuQue.Kb, docs.Id)
		if err != nil {
			log.Println("Get Docs content Err", err.Error())
			continue
		}
		downfilepath := fmt.Sprintf("%s/%s/%d.md", mainpath, article, docs.Id)
		err = downloadDocs(downfilepath, res)
		if err != nil {
			log.Printf("Download Title %s Err: %v", res.Data.Title, err)
		}

	}

	runhugocmd()
	httpwebhook()
}
