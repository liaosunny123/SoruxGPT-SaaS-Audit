package api

import (
	"auditlimit/config"
	"strings"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

func AuditLimit(r *ghttp.Request) {
	ctx := r.Context()

	token := r.Header.Get("Authorization")

	if token != "" {
		token = token[7:]
	}
	g.Log().Debug(ctx, "token", token)

	gfsessionid := r.Cookie.Get("gfsessionid").String()
	g.Log().Debug(ctx, "gfsessionid", gfsessionid)

	referer := r.Header.Get("referer")
	g.Log().Debug(ctx, "referer", referer)

	reqJson, err := r.GetJson()
	if err != nil {
		g.Log().Error(ctx, "GetJson", err)
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"detail": g.Map{
				"message": "Invalid JSON payload.",
			},
		})
	}
	action := reqJson.Get("action").String()
	g.Log().Debug(ctx, "action", action)

	model := reqJson.Get("model").String()
	g.Log().Debug(ctx, "model", model)
	prompt := reqJson.Get("messages.0.content.parts.0").String()
	g.Log().Debug(ctx, "prompt", prompt)

	// 下面开始判断用户状态
	resp, err := g.Client().SetHeaderMap(g.MapStrStr{
		"apiauth":      config.AdminApiKey,
		"Content-Type": "application/json",
	}).Post(ctx, config.ShareUrl+"/adminapi/chatgpt/user/page", g.Map{
		"keyWord": token,
		"page":    1,
		"size":    20,
	})
	if err != nil {
		g.Log().Error(ctx, "GetJson", err)
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"detail": g.Map{
				"message": "Internal Server Error.",
			},
		})
		return
	}
	respJson := gjson.New(resp.ReadAllString())
	if len(respJson.Get("data.list").Array()) == 0 {
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"detail": g.Map{
				"message": "无效的激活码",
			},
		})
		return
	}
	plusMode := respJson.Get("data.list.0.isPlus").Bool()

	if !plusMode && config.PlusModels.Contains(model) {
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"detail": g.Map{
				"message": "普通激活码无法使用 4o 或者是 o1 模型，请使用 gpt-4o-mini",
			},
		})
		return
	}

	// 判断提问内容是否包含禁止词
	if containsAny(ctx, prompt, config.ForbiddenWords) {
		r.Response.Status = 400
		r.Response.WriteJson(g.Map{
			"detail": g.Map{
				"message": "请珍惜账号,不要提问违禁内容.",
			},
		})
		return
	}

	// OPENAI Moderation 检测
	if config.OAIKEY != "" && prompt != "" {
		// 检测是否包含违规内容
		respVar := g.Client().SetHeaderMap(g.MapStrStr{
			"Authorization": "Bearer " + config.OAIKEY,
			"Content-Type":  "application/json",
		}).PostVar(ctx, config.MODERATION, g.Map{
			"input": prompt,
		})

		g.Dump(respVar)
		respJson := gjson.New(respVar)
		isFlagged := respJson.Get("results.0.flagged").Bool()
		g.Log().Debug(ctx, "flagged", isFlagged)
		if isFlagged {
			r.Response.Status = 400
			r.Response.WriteJson(MsgMod400)
			return
		}
	}
	limit, per, limiter, err := GetVisitorWithModel(ctx, token, model)
	if err != nil {
		g.Log().Error(ctx, "GetVisitorWithModel", err)
		r.Response.Status = 500
		r.Response.WriteJson(g.Map{
			"error": err.Error(),
		})
		return
	}
	// 获取剩余次数
	remain := limiter.TokensAt(time.Now())
	g.Log().Debug(ctx, token, model, "remain", remain, "limit", limit, "per", per)
	if remain < 1 {
		r.Response.Status = 429
		reservation := limiter.ReserveN(time.Now(), 1)
		if !reservation.OK() {
			// 处理预留失败的情况，例如返回错误
			r.Response.WriteJson(g.Map{
				"detail": g.Map{
					"message": "You have triggered the usage frequency limit of " + model + ", the current limit is " + gconv.String(limit) + " times/" + gconv.String(per) + ", please wait a moment before trying again.",
				},
				"error": "You have triggered the usage frequency limit of " + model + ", the current limit is " + gconv.String(limit) + " times/" + gconv.String(per) + ", please wait a moment before trying again.\n" + "您已经触发 " + model + " 使用频率限制,当前限制为 " + gconv.String(limit) + " 次/" + gconv.String(per) + ",请稍后再试.",
			})
			reservation.Cancel() // 取消预留，不消耗令牌
			return
		}
		delayFrom := reservation.Delay()
		reservation.Cancel() // 取消预留，不消耗令牌

		g.Log().Debug(ctx, "delayFrom", delayFrom)
		r.Response.WriteJson(g.Map{
			"error": "You have triggered the usage frequency limit of " + model + ", the current limit is " + gconv.String(limit) + " times/" + gconv.String(per) + ", please wait " + gconv.String(int(delayFrom.Seconds())) + " seconds before trying again.\n" + "您已经触发 " + model + " 使用频率限制,当前限制为 " + gconv.String(limit) + " 次/" + gconv.String(per) + ",请等待 " + gconv.String(int(delayFrom.Seconds())) + " 秒后再试.",
			"detail": g.Map{
				"message": "You have triggered the usage frequency limit of " + model + ", the current limit is " + gconv.String(limit) + " times/" + gconv.String(per) + ", please wait " + gconv.String(int(delayFrom.Seconds())) + " seconds before trying again.",
			},
		})
		return
	}
	// 消耗一个令牌
	limiter.Allow()

	r.Response.Status = 200

}

// 判断字符串是否包含数组中的任意一个元素
func containsAny(ctx g.Ctx, text string, array []string) bool {
	for _, item := range array {
		if strings.Contains(text, item) {
			g.Log().Debug(ctx, "containsAny", text, item)
			return true
		}
	}
	return false
}
