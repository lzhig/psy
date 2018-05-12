package robot

import (
	"math/rand"
	"time"

	"github.com/lzhig/rapidgo/base"
)

// ActionID 行动ID
type ActionID int

// Robot 机器人
type Robot struct {
	drivers map[string]*RobotDriver
	driver  *RobotDriver
}

// Init 初始化
func (obj *Robot) Init() {
	obj.drivers = make(map[string]*RobotDriver)
}

// Set 设置
func (obj *Robot) Set(name string, driver *RobotDriver) {
	obj.drivers[name] = driver
}

// SwitchDriver 切换执行的驱动器
func (obj *Robot) SwitchDriver(name string) bool {
	if driver, ok := obj.drivers[name]; ok {
		obj.driver = driver
		return true
	}
	return false
}

// DoAction 机器人行动
func (obj *Robot) DoAction(actionID ActionID) {
	if obj.driver == nil {
		base.LogError("No driver in robot.")
		return
	}

	obj.driver.Drive(actionID)
}

// RobotDriver 机器人驱动器
type RobotDriver struct {
	strategies map[ActionID]*RobotStrategy
}

// NewRobotDriver 创建RobotDriver
func NewRobotDriver() *RobotDriver {
	return &RobotDriver{
		strategies: make(map[ActionID]*RobotStrategy),
	}
}

// Init 初始化
func (obj *RobotDriver) Init() {
	obj.strategies = make(map[ActionID]*RobotStrategy)
}

// Set 设置
func (obj *RobotDriver) Set(actionID ActionID, strategy *RobotStrategy) {
	obj.strategies[actionID] = strategy
}

// Drive 驱动
func (obj *RobotDriver) Drive(actionID ActionID) {
	if strategy, ok := obj.strategies[actionID]; ok {
		strategy.Do()
	}
}

// RobotAction 行动
type RobotAction struct {
	rate   int32
	action func()
}

// NewRobotAction 创建RobotAction
func NewRobotAction(rate int32, action func()) *RobotAction {
	return &RobotAction{rate: rate, action: action}
}

// RobotStrategy 机器人策略
type RobotStrategy struct {
	strategies []*RobotAction
	totalRate  int32
	s          *rand.Rand
}

// Set 设置
func (obj *RobotStrategy) Set(strategies []*RobotAction) {
	obj.strategies = strategies
	obj.s = rand.New(rand.NewSource(time.Now().UnixNano()))
	obj.totalRate = 0
	for _, action := range obj.strategies {
		obj.totalRate += action.rate
	}
}

// Do 执行
func (obj *RobotStrategy) Do() {
	action := obj.getRandomAction()
	if action == nil {
		base.LogError("Failed to get a random action")
		return
	}

	action.action()
}

func (obj *RobotStrategy) getRandomAction() *RobotAction {
	if obj.totalRate <= 0 {
		return nil
	}

	r := obj.s.Int31n(obj.totalRate)
	for _, action := range obj.strategies {
		if r < action.rate {
			return action
		}

		r -= action.rate
	}

	return nil
}
