package service

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xflash-panda/server-shadowsocks/internal/pkg/api"
	cProtocol "github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/task"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/proxy"
	"time"
)

type Config struct {
	SysInterval time.Duration
}

type Builder struct {
	instance                *core.Instance
	config                  *Config
	nodeInfo                *api.NodeInfo
	inboundTag              string
	userList                []*api.UserInfo
	getUserList             func() ([]*api.UserInfo, error)
	reportUserTraffic       func([]*api.UserTraffic) error
	nodeInfoMonitorPeriodic *task.Periodic
	userReportPeriodic      *task.Periodic
}

// New return a builder service with default parameters.
func New(inboundTag string, instance *core.Instance, config *Config, nodeInfo *api.NodeInfo,
	getUserList func() ([]*api.UserInfo, error), reportUserTraffic func([]*api.UserTraffic) error,
) *Builder {
	builder := &Builder{
		inboundTag:        inboundTag,
		instance:          instance,
		config:            config,
		nodeInfo:          nodeInfo,
		getUserList:       getUserList,
		reportUserTraffic: reportUserTraffic,
	}
	return builder
}

// Start implement the Start() function of the service interface
func (b *Builder) Start() error {
	// Update user
	userList, err := b.getUserList()
	if err != nil {
		return err
	}
	err = b.addNewUser(userList, b.nodeInfo)
	if err != nil {
		return err
	}

	b.userList = userList

	b.nodeInfoMonitorPeriodic = &task.Periodic{
		Interval: b.config.SysInterval,
		Execute:  b.nodeInfoMonitor,
	}
	b.userReportPeriodic = &task.Periodic{
		Interval: b.config.SysInterval,
		Execute:  b.userInfoMonitor,
	}
	log.Infoln("Start monitor node status")
	err = b.nodeInfoMonitorPeriodic.Start()
	if err != nil {
		return fmt.Errorf("node info periodic, start erorr:%s", err)
	}
	log.Infoln("Start report node status")
	err = b.userReportPeriodic.Start()
	if err != nil {
		return fmt.Errorf("user report periodic, start erorr:%s", err)
	}
	return nil
}

// Close implement the Close() function of the service interface
func (b *Builder) Close() error {
	if b.nodeInfoMonitorPeriodic != nil {
		err := b.nodeInfoMonitorPeriodic.Close()
		if err != nil {
			return fmt.Errorf("node info periodic close failed: %s", err)
		}
	}

	if b.nodeInfoMonitorPeriodic != nil {
		err := b.userReportPeriodic.Close()
		if err != nil {
			return fmt.Errorf("user report periodic close failed: %s", err)
		}
	}
	return nil
}

//addNewUser
func (b *Builder) addNewUser(userInfo []*api.UserInfo, nodeInfo *api.NodeInfo) (err error) {
	users := make([]*cProtocol.User, 0)
	users = buildUser(b.inboundTag, userInfo, nodeInfo.Cipher)
	err = b.addUsers(users, b.inboundTag)
	if err != nil {
		return err
	}
	log.Printf("Added %d new users", len(userInfo))
	return nil
}

//add Users
func (b *Builder) addUsers(users []*cProtocol.User, tag string) error {
	inboundManager := b.instance.GetFeature(inbound.ManagerType()).(inbound.Manager)
	handler, err := inboundManager.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.UserManager", err)
	}
	for _, item := range users {
		mUser, err := item.ToMemoryUser()
		if err != nil {
			return err
		}
		err = userManager.AddUser(context.Background(), mUser)
		if err != nil {
			return err
		}
	}
	return nil
}

//removeUsers
func (b *Builder) removeUsers(users []string, tag string) error {
	inboundManager := b.instance.GetFeature(inbound.ManagerType()).(inbound.Manager)
	handler, err := inboundManager.GetHandler(context.Background(), tag)
	if err != nil {
		return fmt.Errorf("no such inbound tag: %s", err)
	}
	inboundInstance, ok := handler.(proxy.GetInbound)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.GetInbound", tag)
	}

	userManager, ok := inboundInstance.GetInbound().(proxy.UserManager)
	if !ok {
		return fmt.Errorf("handler %s is not implement proxy.UserManager", err)
	}
	for _, email := range users {
		err = userManager.RemoveUser(context.Background(), email)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) getTraffic(email string) (up int64, down int64) {
	upName := "user>>>" + email + ">>>traffic>>>uplink"
	downName := "user>>>" + email + ">>>traffic>>>downlink"
	statsManager := b.instance.GetFeature(stats.ManagerType()).(stats.Manager)
	upCounter := statsManager.GetCounter(upName)
	downCounter := statsManager.GetCounter(downName)
	if upCounter != nil {
		up = upCounter.Value()
		upCounter.Set(0)
	}
	if downCounter != nil {
		down = downCounter.Value()
		downCounter.Set(0)
	}
	return up, down

}

//userInfoMonitor
func (b *Builder) userInfoMonitor() (err error) {
	// Get User traffic
	userTraffic := make([]*api.UserTraffic, 0)
	for _, user := range b.userList {
		up, down := b.getTraffic(buildUserEmail(b.inboundTag, user.ID, user.UUID))
		if up > 0 || down > 0 {
			userTraffic = append(userTraffic, &api.UserTraffic{
				UID:      user.ID,
				Upload:   up,
				Download: down})
		}
	}

	log.Infof("%d user traffic needs to be reported", len(userTraffic))
	if len(userTraffic) > 0 {
		err = b.reportUserTraffic(userTraffic)
		if err != nil {
			log.Errorln(err)
		}
	}

	return nil
}

func (b *Builder) nodeInfoMonitor() (err error) {
	// Update User
	newUserInfo, err := b.getUserList()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	deleted, added := compareUserList(b.userList, newUserInfo)
	if len(deleted) > 0 {
		deletedEmail := make([]string, len(deleted))
		for i, u := range deleted {
			deletedEmail[i] = buildUserEmail(b.inboundTag, u.ID, u.UUID)
		}
		err := b.removeUsers(deletedEmail, b.inboundTag)
		if err != nil {
			log.Print(err)
		}
	}
	if len(added) > 0 {
		err = b.addNewUser(added, b.nodeInfo)
		if err != nil {
			log.Errorln(err)
		}

	}
	log.Infof("%d user deleted, %d user added", len(deleted), len(added))
	b.userList = newUserInfo
	return nil
}

//compareUserList
func compareUserList(old, new []*api.UserInfo) (deleted, added []*api.UserInfo) {
	msrc := make(map[*api.UserInfo]byte) //?????????????????????
	mall := make(map[*api.UserInfo]byte) //???+????????????????????????

	var set []*api.UserInfo //??????

	//1.???????????????map
	for _, v := range old {
		msrc[v] = 0
		mall[v] = 0
	}
	//2.???????????????????????????????????????????????????????????????????????????????????????
	for _, v := range new {
		l := len(mall)
		mall[v] = 1
		if l != len(mall) { //???????????????????????????
			l = len(mall)
		} else { //?????????????????????
			set = append(set, v)
		}
	}
	//3.??????????????????????????????????????????????????????????????????????????????????????????-???=????????????????????????
	for _, v := range set {
		delete(mall, v)
	}
	//4.?????????mall????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????
	for v := range mall {
		_, exist := msrc[v]
		if exist {
			deleted = append(deleted, v)
		} else {
			added = append(added, v)
		}
	}

	return deleted, added
}
