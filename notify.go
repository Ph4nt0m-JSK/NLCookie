package jd_cookie

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cdle/sillyGirl/core"
	"github.com/cdle/sillyGirl/develop/qinglong"
	cron "github.com/robfig/cron/v3"
)

type JdNotify struct {
	ID           string
	Pet          bool
	Fruit        bool
	DreamFactory bool
	Note         string
	PtKey        string
	AssetCron    string
}

var cc *cron.Cron

var jdNotify = core.NewBucket("jdNotify")

func assetPush(pt_pin string) {
	jn := &JdNotify{
		ID: pt_pin,
	}
	jdNotify.First(jn)
	qqGroup := jd_cookie.GetInt("qqGroup")
	if jn.PtKey != "" && pt_pin != "" {
		pt_key := jn.PtKey
		for _, tp := range []string{
			"qq", "tg", "wx", "wxmp",
		} {
			var fs []func()
			core.Bucket("pin" + strings.ToUpper(tp)).Foreach(func(k, v []byte) error {
				if string(k) == pt_pin && pt_pin != "" {
					if push, ok := core.Pushs[tp]; ok {
						fs = append(fs, func() {
							push(string(v), GetAsset(&JdCookie{
								PtPin: pt_pin,
								PtKey: pt_key,
							}), qqGroup)
						})
					}
				}
				return nil
			})
			if len(fs) != 0 {
				for _, f := range fs {
					f()
				}
			}
		}
	}
}

func initNotify() {
	var ccc = map[string]cron.EntryID{}
	cc = cron.New(cron.WithSeconds())
	cc.Start()
	jdNotify.Foreach(func(_, v []byte) error {
		aa := &JdNotify{}
		json.Unmarshal(v, aa)
		if aa.AssetCron != "" {
			if rid, err := cc.AddFunc(aa.AssetCron, func() {
				assetPush(aa.ID)
			}); err == nil {
				ccc[aa.ID] = rid
			}
		}
		return nil
	})
	go func() {
		for {
			time.Sleep(time.Second * 2)
			envs, _ := qinglong.GetEnvs("JD_COOKIE")
			for _, env := range envs {
				if env.Status != 0 {
					continue
				}
				pt_pin := core.FetchCookieValue(env.Value, "pt_pin")
				pt_key := core.FetchCookieValue(env.Value, "pt_key")
				if pt_pin != "" && pt_key != "" {
					jn := &JdNotify{
						ID: pt_pin,
					}
					jdNotify.First(jn)
					if jn.PtKey != pt_key {
						jn.PtKey = pt_key
						jdNotify.Create(jn)
					}
				}
			}
		}
	}()
	core.AddCommand("", []core.Function{
		{
			Rules: []string{`raw ^任务通知$`},
			Cron:  jd_cookie.Get("task_Notify", "0 12 * * *"),
			Admin: true,
			Handle: func(_ core.Sender) interface{} {
				envs, _ := qinglong.GetEnvs("JD_COOKIE")
				for _, env := range envs {
					initPetTown(env.Value, nil)
					initFarm(env.Value, nil)
					dream(env.Value, nil)
				}
				return "白嫖榜单推送完成~~~"
			},
		},
		{
			Rules: []string{`raw ^关闭(.+)通知$`},
			Handle: func(s core.Sender) interface{} {
				class := s.Get()
				pin := pin(s.GetImType())
				uid := fmt.Sprint(s.GetUserID())
				accounts := []string{}
				pin.Foreach(func(k, v []byte) error {
					if string(v) == uid {
						accounts = append(accounts, string(k))
					}
					return nil
				})
				for i := range accounts {
					jn := &JdNotify{
						ID: accounts[i],
					}
					jdNotify.First(jn)
					if class == "京喜工厂" {
						jn.DreamFactory = true
					}
					if class == "东东农场" {
						jn.Fruit = true
					}
					if class == "东东萌宠" {
						jn.Pet = true
					}
					jdNotify.Create(jn)
				}
				return fmt.Sprintf("佩琦已为你关闭%d个账号的"+class+"通知~~~", len(accounts))
			},
		},
		{
			Rules: []string{`raw ^账号管理$`},
			Handle: func(s core.Sender) interface{} {
				if groupCode := jd_cookie.Get("groupCode"); !s.IsAdmin() && groupCode != "" && s.GetChatID() != 0 && !strings.Contains(groupCode, fmt.Sprint(s.GetChatID())) {
					s.Continue()
					return nil
				}
				pin := pin(s.GetImType())
				uid := fmt.Sprint(s.GetUserID())
				accounts := []string{}
				pin.Foreach(func(k, v []byte) error {
					if string(v) == uid {
						accounts = append(accounts, string(k))
					}
					return nil
				})
				num := len(accounts)
				if num == 0 {
					return "抱歉，你尚未绑定的账号~~~\n请私聊我发送命令：登录，进行登录~~~\n或添加管理员微信：Lin-VowNight，进行人工登录~~~"
				}
				ask := fmt.Sprintf("请在20秒内从1~%d中选择你要操作的账号：\n", num)
				for i := range accounts {
					jn := &JdNotify{
						ID: accounts[i],
					}
					jdNotify.First(jn)
					note := ""
					if jn.Note != "" {
						note = jn.Note
					} else {
						note = jn.ID
					}
					ask += fmt.Sprintf("%d. %s\n", i+1, note)
				}
				s.Reply(strings.Trim(ask, "\n"))
				rt := s.Await(s, func(s core.Sender) interface{} {
					return core.Range([]int{1, num})
				}, time.Second*20)
				switch rt.(type) {
				case nil:
					return "操作超时，已退出账号管理会话~~~"
				case int:
					pt_pin := accounts[rt.(int)-1]
					jn := &JdNotify{
						ID: pt_pin,
					}
					jdNotify.First(jn)
					ask := "请在20秒内选择操作：\n1. 查询京东账号资产\n"

					if jn.Note == "" {
						ask += "2. 添加账户备注信息\n"
					} else {
						ask += "2. 修改账户备注信息\n"
					}
					if jn.Pet {
						ask += "3. 开启东东萌宠通知\n"
					} else {
						ask += "3. 关闭东东萌宠通知\n"
					}
					if jn.Fruit {
						ask += "4. 开启东东果园通知\n"
					} else {
						ask += "4. 关闭东东果园通知\n"
					}
					if jn.DreamFactory {
						ask += "5. 开启京喜工厂通知\n"
					} else {
						ask += "5. 关闭京喜工厂通知\n"
					}
					if jn.AssetCron == "" {
						ask += "6. 添加资产推送时间\n"
					} else {
						ask += "6. 修改资产推送时间\n"
					}
					ask += "7. 解绑当前京东账号\n8. 退出当前管理会话"
					s.Reply(ask)
					rt := s.Await(s, func(s core.Sender) interface{} {
						return core.Range([]int{1, 8})
					}, time.Second*20)
					switch rt.(type) {
					case nil:
						return "操作超时，已退出账号管理会话~~~"
					case int:
						switch rt.(int) {
						case 1:
							if jn.PtKey == "" {
								return "提醒：该🐶京东账号已过期~~~\n请重新私聊我发送命令：登录，进行登录~~~\n或添加管理员微信：Lin-VowNight，进行人工登录~~~\n\n本通知连续出现二次以上才是真正的帐号失效!!!"
							}
							return GetAsset(&JdCookie{
								PtPin: pt_pin,
								PtKey: jn.PtKey,
							})
						case 2:
							s.Reply("请输入新的账号备注信息：")
							jn.Note = s.Await(s, nil).(string)
						case 3:
							jn.Pet = !jn.Pet
						case 4:
							jn.Fruit = !jn.Fruit
						case 5:
							jn.DreamFactory = !jn.DreamFactory
						case 6:
							s.Reply("请输入资产推送时间(格式00:00:00，对应时、分、秒):")
							rt := s.Await(s, nil).(string)
							_, err := time.ParseInLocation("2006-01-02 15:04:05", time.Now().Format("2006-01-02"+" ")+rt, time.Local)
							if err != nil {
								s.Reply("时间格式错误，已退出设置定时推送会话~~~")
								return nil
							}
							dd := strings.Split(rt, ":")
							jn.AssetCron = fmt.Sprintf("%s %s %s * * *", dd[2], dd[1], dd[0])
							if rid, ok := ccc[jn.ID]; ok {
								cc.Remove(rid)
								if rid, err := cc.AddFunc(jn.AssetCron, func() {
									assetPush(jn.ID)
								}); err == nil {
									ccc[jn.ID] = rid
								} else {
									return err
								}
							}
						case 7:
							pin.Set(pt_pin, "")
							return "🐶京东账号解绑成功，账号管理会话结束~~~"
						case 8:
							return "已退出账号管理会话~~~"
						}
					}
					jdNotify.Create(jn)
					return "通知操作成功，通知管理会话结束~~~"
				}
				return nil
			},
		},
	})
}

func a叉哦叉哦(pt_pin, class, content string) {
	u := &JdNotify{
		ID: pt_pin,
	}
	jdNotify.First(u)
	if u.DreamFactory && class == "京喜工厂" {
		return
	}
	if u.Fruit && class == "东东农场" {
		return
	}
	if u.Pet && class == "东东萌宠" {
		return
	}
	if u.Note == "" {
		u.Note = u.ID
	}
	Notify(pt_pin, class+"通知("+u.Note+")：\n"+content+"\n\n可发送命令：账号管理，根据菜单自定义设置你的账号通知信息~~~")
}
