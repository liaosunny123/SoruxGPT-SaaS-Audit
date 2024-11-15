package main

import (
	"auditlimit/api"
	"auditlimit/config"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
)

func main() {
	s := g.Server()
	s.SetPort(config.PORT)
	s.BindHandler("/", Index)
	s.BindHandler("/audit_limit", api.AuditLimit)
	s.Run()
}

func Index(r *ghttp.Request) {
	r.Response.Write("Hello SoruxGPT SaaS, this is the audit limit target for SoruxGPT SaaS.")
}

func init() {
	if gfile.Exists("./data/keywords.txt") {
		keyWords := gfile.GetContents("./data/keywords.txt")
		keyWordsSlice := gstr.Split(keyWords, "\n")
		if len(keyWordsSlice) > 0 {
			for i := 0; i < len(keyWordsSlice); i++ {
				keyWordsSlice[i] = gstr.Trim(keyWordsSlice[i])
				if keyWordsSlice[i] == "" {
					keyWordsSlice = append(keyWordsSlice[:i], keyWordsSlice[i+1:]...)
					i--
				}
			}
		}
		config.ForbiddenWords = keyWordsSlice
	}
}
