/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 10:10:59
 * @modify date 2018-01-19 10:10:59
 * @desc [description]
 */

package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
)

type serverConfig struct {
	Addr         string `json:"addr"`          // 服务器地址
	MaxUsers     uint32 `json:"max_users"`     // 最大用户数
	IdleTime     uint32 `json:"idle_time"`     // 最大闲置时间,单位s
	LoginTimeout uint32 `json:"login_timeout"` // 登录超时,单位s
	Gm           string `json:"gm"`            // gm地址
}

type mysqlConfig struct {
	Addr     string `json:"addr"`      // mysql服务器地址
	Username string `json:"username"`  // 用户名
	Password string `json:"password"`  // 密码
	Db       string `json:"db"`        // 数据库名
	MaxConns int    `json:"max_conns"` // 最大连接数
}

type roomConfig struct {
	RoomNameLen                  int      `json:"room_name_len"`                     // 房间名称的最大长度
	CreditPoints                 []uint32 `json:"credit_points"`                     // credit points
	RoomRate                     uint32   `json:"room_rate"`                         // 房费，一局多少钻石
	CountCreated                 uint32   `json:"count_created"`                     // 每个玩家创建存在的房间数
	MaxTablePlayers              uint32   `json:"max_table_players"`                 // 每个桌子最大入座玩家数
	DealCardsNum                 uint32   `json:"deal_cards_num"`                    // 每个玩家的发牌数
	MaxPlayers                   uint32   `json:"max_players"`                       // 一个房间最大玩家数
	MaxBetRate                   uint32   `json:"max_bet_rate"`                      // 最大下注是最小下注的倍数
	StatesCountdown              []uint32 `json:"states_countdown"`                  // 游戏阶段的超时时间，0为不过期
	ScoreboardCountPerTime       uint32   `json:"scoreboard_count_per_time"`         // 每次请求返回积分榜的最大条数
	KickNoBetForHands            uint32   `json:"kick_no_bet_for_hands"`             // 踢离连续没有下注的局数
	ReleaseTimeoutSec            uint32   `json:"release_timeout_sec"`               // 房间对象释放的超时时间
	CareerRoomRecordCountPerTime uint32   `json:"career_room_record_count_per_time"` // 每次请求返回生涯房间记录的最大条数
}

type userConfig struct {
	FacebookAvatarType string `json:"facebook_avatar_type"` // facebook用户头像的类型, large, normal, ...
}

type diamondsConfig struct {
	InitDiamonds    uint32  `json:"init_diamonds"`     // 初始钻石，用户注册时
	SendDiamondsFee float64 `json:"send_diamonds_fee"` // 发送钻石的手续费
}

type statisticConfig struct {
	DateSheet      string `json:"date_sheet"`      // 每天开始统计报表的时间
	OnlineFilename string `json:"online_filename"` // 在线统计存档文件
}

type versionServiceConfig struct {
	Addr string `json:"addr"` // 版本更新服务地址
	File string `json:"file"` // 版本信息文件
}

type noticeConfig struct {
	File string `json:"file"` // 公告配置文件
}

// Config type
type Config struct {
	Version        string               `json:"version"`         // 版本号
	Debug          bool                 `json:"debug"`           // 调试模式
	CPUNum         int                  `json:"cpu_num"`         // 使用的CPU参数
	Server         serverConfig         `json:"server"`          // 服务器配置
	Mysql          mysqlConfig          `json:"mysql"`           // mysql配置
	Room           roomConfig           `json:"room"`            // 房间相关配置
	User           userConfig           `json:"user"`            // 用户相关配置
	Diamonds       diamondsConfig       `json:"diamonds"`        // 钻石相关配置
	Statistic      statisticConfig      `json:"statistic"`       // 统计相关配置
	VersionService versionServiceConfig `json:"version_service"` // 版本更新服务相关配置
	Notice         noticeConfig         `json:"notice"`          // 公告配置
}

// Load function
func (obj *Config) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, obj)
}
