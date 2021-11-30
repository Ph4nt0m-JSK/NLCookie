package jd_cookie

import (
	"fmt"
	"strings"
	"time"

	"github.com/cdle/sillyGirl/core"
	"github.com/cdle/sillyGirl/develop/qinglong"
	"github.com/gin-gonic/gin"
)

var pinQQ = core.NewBucket("pinQQ")
var pinTG = core.NewBucket("pinTG")
var pinWXMP = core.NewBucket("pinWXMP")
var pinWX = core.NewBucket("pinWX")
var pin = func(class string) core.Bucket {
	return core.Bucket("pin" + strings.ToUpper(class))
}

func initSubmit() {
	//
	core.Server.POST("/cookie", func(c *gin.Context) {
		cookie := c.Query("ck")
		ck := &JdCookie{
			PtKey: core.FetchCookieValue(cookie, "pt_key"),
			PtPin: core.FetchCookieValue(cookie, "pt_pin"),
		}
		type Result struct {
			Code    int         `json:"code"`
			Data    interface{} `json:"data"`
			Message string      `json:"message"`
		}
		result := Result{
			Data: nil,
			Code: 300,
		}
		if ck.PtPin == "" || ck.PtKey == "" {
			result.Message = "一句mmp，不知当讲不当讲~~~"
			c.JSON(200, result)
			return
		}
		if !ck.Available() {
			result.Message = "无效的ck，请确认🐶京东账号格式正确，不允许带有空格~~~\n如还出现此提示，请前往京东app修改🐶京东昵称\n或请私聊我发送命令：登录，进行登录~~~\n也可添加管理员微信：Lin-VowNight，进行人工登录绑定你的🐶京东账号信息~~~"
			c.JSON(200, result)
			return
		}
		value := fmt.Sprintf(`pt_key=%s;\s*?pt_pin=%s;`, ck.PtKey, ck.PtPin)
		envs, err := qinglong.GetEnvs("JD_COOKIE")
		if err != nil {
			result.Message = err.Error()
			c.JSON(200, result)
			return
		}
		find := false
		for _, env := range envs {
			if strings.Contains(env.Value, fmt.Sprintf("pt_pin=%s;", ck.PtPin)) {
				envs = []qinglong.Env{env}
				find = true
				break
			}
		}
		if !find {

			if err := qinglong.AddEnv(qinglong.Env{
				Name:  "JD_COOKIE",
				Value: value,
			}); err != nil {
				result.Message = err.Error()
				c.JSON(200, result)
				return
			}
			rt := "🐶京东账号：" + ck.Nickname + "，已添加成功~~~"
			core.NotifyMasters(rt)
			result.Message = rt
			result.Code = 200
			c.JSON(200, result)
			return
		} else {
			env := envs[0]
			env.Value = value
			if env.Status != 0 {
				if err := qinglong.Config.Req(qinglong.PUT, qinglong.ENVS, "/enable", []byte(`["`+env.ID+`"]`)); err != nil {
					result.Message = err.Error()
					c.JSON(200, result)
					return
				}
				env.Status = 0
				if err := qinglong.UdpEnv(env); err != nil {
					result.Message = err.Error()
					c.JSON(200, result)
					return
				}
			}
			rt := "🐶京东账号：" + ck.Nickname + "，已更新成功~~~"
			core.NotifyMasters(rt)
			result.Message = rt
			result.Code = 200
			c.JSON(200, result)
			return
		}
	})
	core.AddCommand("jd", []core.Function{
		// {
		// 	Rules: []string{`unbind ?`},
		// 	Admin: true,
		// 	Handle: func(s core.Sender) interface{} {
		// 		s.Disappear(time.Second * 40)
		// 		envs, err := qinglong.GetEnvs("JD_COOKIE")
		// 		if err != nil {
		// 			return err
		// 		}
		// 		if len(envs) == 0 {
		// 			return "暂时无法进行解绑操作~~~"
		// 		}
		// 		key := s.Get()
		// 		pin := pin(s.GetImType())
		// 		for _, env := range envs {
		// 			pt_pin := FetchJdCookieValue("pt_pin", env.Value)
		// 			pin.Foreach(func(k, v []byte) error {
		// 				if string(k) == pt_pin && string(v) == key {
		// 					s.Reply(fmt.Sprintf("🐶京东账号：%s，已成功解绑~~~", pt_pin))
		// 					pin.Set(string(k), "")
		// 				}
		// 				return nil
		// 			})
		// 		}
		// 		return "解绑操作已完成~~~"
		// 	},
		// },
		{
			Rules: []string{"send ? ?"},
			Admin: true,
			Handle: func(s core.Sender) interface{} {
				user_pin := s.Get()
				msg := s.Get(1)
				for _, tp := range []string{
					"qq", "tg", "wx", "wxmp",
				} {
					core.Bucket("pin" + strings.ToUpper(tp)).Foreach(func(k, v []byte) error {
						pt_pin := string(k)
						if pt_pin == user_pin || user_pin == "all" {
							if push, ok := core.Pushs[tp]; ok {
								push(string(v), msg, nil)
							}
						}
						return nil
					})
				}
				return "消息已成功发送~~~"
			},
		},
		{
			Rules: []string{`unbind`},
			Handle: func(s core.Sender) interface{} {
				s.Disappear(time.Second * 40)

				uid := fmt.Sprint(s.GetUserID())

				pin := pin(s.GetImType())
				pin.Foreach(func(k, v []byte) error {
					if string(v) == uid {
						s.Reply(fmt.Sprintf("🐶京东账号：%s，已成功解绑~~~", string(k)))
						pin.Set(string(k), "")
					}
					return nil
				})
				return "解绑操作已完成~~~"
			},
		},
		{
			Rules:   []string{`raw pt_key=([^;=\s]+);\s*pt_pin=([^;=\s]+)`},
			FindAll: true,
			Handle: func(s core.Sender) interface{} {
				s.Reply(s.Delete())
				s.Disappear(time.Second * 20)
				for _, v := range s.GetAllMatch() {
					ck := &JdCookie{
						PtKey: v[0],
						PtPin: v[1],
					}
					if len(ck.PtKey) <= 20 {
						s.Reply("再捣乱我就报警啦！") //
						continue
					}
					if !ck.Available() {
						s.Reply("无效的🐶京东账号，有瞎编ck的嫌疑~~~") //有瞎编ck的嫌疑
						continue
					}
					if ck.Nickname == "" {
						s.Reply("无效的🐶京东昵称，请确认🐶京东账号格式正确，不允许带有空格~~~\n如还出现此提示，请前往京东app修改🐶京东昵称\n或请私聊我发送命令：登录，进行登录~~~\n也可添加管理员微信：Lin-VowNight，进行人工登录绑定你的🐶京东账号信息~~~")
					}

					value := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
					if jd_cookie.Get("xdd_url") != "" {
						if s.GetImType() == "qq" {
							xdd(value, fmt.Sprint(s.GetUserID()))
						} else {
							s.Reply("需要QQ推送资产信息的朋友\n请在30秒内输入您的QQ号进行绑定：")
							s.Await(s, func(s core.Sender) interface{} {
								xdd(value, s.GetContent())
								return "OK"
							}, `^\d+$`, time.Second*30)
						}
					}
					envs, err := qinglong.GetEnvs("JD_COOKIE")
					if err != nil {
						s.Reply(err)
						continue
					}
					find := false
					for _, env := range envs {
						if strings.Contains(env.Value, fmt.Sprintf("pt_pin=%s;", ck.PtPin)) {
							envs = []qinglong.Env{env}
							find = true
							break
						}
					}
					pin(s.GetImType()).Set(ck.PtPin, s.GetUserID())
					if !find {
						if err := qinglong.AddEnv(qinglong.Env{
							Name:  "JD_COOKIE",
							Value: value,
						}); err != nil {
							s.Reply(err)
							continue
						}
						rt := "🐶京东账号：" + ck.Nickname + "，已添加成功~~~"
						core.NotifyMasters(rt)
						s.Reply(rt)
						continue
					} else {
						env := envs[0]
						env.Value = value
						if env.Status != 0 {
							if err := qinglong.Config.Req(qinglong.PUT, qinglong.ENVS, "/enable", []byte(`["`+env.ID+`"]`)); err != nil {
								s.Reply(err)
								continue
							}
						}
						env.Status = 0
						if err := qinglong.UdpEnv(env); err != nil {
							s.Reply(err)
							continue
						}
						assets.Delete(ck.PtPin)
						rt := "🐶京东账号：" + ck.Nickname + "，已更新成功~~~"
						core.NotifyMasters(rt)
						s.Reply(rt)
						continue
					}
				}
				return nil
			},
		},
		{
			Rules:   []string{`raw pin=([^;=\s]+);\s*wskey=([^;=\s]+)`},
			FindAll: true,
			Handle: func(s core.Sender) interface{} {
				s.Reply(s.Delete())
				s.Disappear(time.Second * 20)
				value := fmt.Sprintf("pin=%s;wskey=%s;", s.Get(0), s.Get(1))

				pt_key, err := getKey(value)
				if err == nil {
					if strings.Contains(pt_key, "fake") {
						return "无效的wskey，请确认🐶京东账号格式正确，不允许带有空格~~~\n如还出现此提示，请前往京东app修改🐶京东昵称\n或请私聊我发送命令：登录，进行登录~~~\n也可添加管理员微信：Lin-VowNight，进行人工登录绑定你的🐶京东账号信息~~~"
					}
				} else {
					s.Reply(err)
				}
				ck := &JdCookie{
					PtKey: pt_key,
					PtPin: s.Get(0),
				}
				ck.Available()
				envs, err := qinglong.GetEnvs("pin=")
				if err != nil {
					return err
				}
				pin(s.GetImType()).Set(ck.PtPin, s.GetUserID())
				var envCK *qinglong.Env
				var envWsCK *qinglong.Env
				for i := range envs {
					if strings.Contains(envs[i].Value, fmt.Sprintf("pin=%s;wskey=", ck.PtPin)) && envs[i].Name == "JD_WSCK" {
						envWsCK = &envs[i]
					} else if strings.Contains(envs[i].Value, fmt.Sprintf("pt_pin=%s;", ck.PtPin)) && envs[i].Name == "JD_COOKIE" {
						envCK = &envs[i]
					}
				}
				value2 := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
				if envCK == nil {
					qinglong.AddEnv(qinglong.Env{
						Name:  "JD_COOKIE",
						Value: value2,
					})
				} else {
					if envCK.Status != 0 {
						envCK.Value = value2
						if err := qinglong.UdpEnv(*envCK); err != nil {
							return err
						}
					}
				}
				if envWsCK == nil {
					if err := qinglong.AddEnv(qinglong.Env{
						Name:  "JD_WSCK",
						Value: value,
					}); err != nil {
						return err
					}
					return "🐶京东账号：" + ck.Nickname + "，已添加成功~~~"
				} else {
					envWsCK.Value = value
					if envWsCK.Status != 0 {
						if err := qinglong.Config.Req(qinglong.PUT, qinglong.ENVS, "/enable", []byte(`["`+envWsCK.ID+`"]`)); err != nil {
							return err
						}
					}
					envWsCK.Status = 0
					if err := qinglong.UdpEnv(*envWsCK); err != nil {
						return err
					}
					return "🐶京东账号：" + ck.Nickname + "，已更新成功~~~"
				}
			},
		},
	})
}
