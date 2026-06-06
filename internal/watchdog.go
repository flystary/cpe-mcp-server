package internal

import (
	"log"
	"sync"
	"time"
)

// Rollable 只要南向驱动实现了这两个方法，即代表它“需要并具备”回退能力
type Rollable interface {
	Save(target string) error
	Roll(target string) error
}

type ConfigTransactionManager struct {
	mu            sync.Mutex
	watchdogTimer *time.Timer
	isPending     bool
	activeDriver  Rollable
	activeTarget  string
}

var globalTxManager = &ConfigTransactionManager{}

func GetTxManager() *ConfigTransactionManager { return globalTxManager }

// RegisterAndExecute 强类型接收：省去运行期反射，编译期不满足 Rollable 直接拒绝编译
func (tm *ConfigTransactionManager) RegisterAndExecute(driver Rollable, target string, executeFn func() error) error {
	tm.mu.Lock()

	log.Printf("[Watchdog] 接收到高保底强类型驱动，自动拉起两阶段配置保护...")

	if err := driver.Save(target); err != nil {
		tm.mu.Unlock()
		return err
	}

	if tm.isPending && tm.watchdogTimer != nil {
		tm.watchdogTimer.Stop()
	}

	tm.isPending = true
	tm.activeDriver = driver
	tm.activeTarget = target

	tm.watchdogTimer = time.AfterFunc(45*time.Second, func() {
		tm.mu.Lock()
		defer tm.mu.Unlock()
		if tm.isPending && tm.activeDriver != nil {
			log.Println("[CRITICAL_WATCHDOG] 确认超时！强行触发内核状态 Rollback...")
			_ = tm.activeDriver.Roll(tm.activeTarget)
			tm.isPending = false
			tm.activeDriver = nil
		}
	})
	tm.mu.Unlock()

	return executeFn()
}

func (tm *ConfigTransactionManager) ConfirmCommit() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.isPending {
		if tm.watchdogTimer != nil {
			tm.watchdogTimer.Stop()
		}
		tm.isPending = false
		tm.activeDriver = nil
		log.Println("[Watchdog] 连通性通过，安全释放内存旧快照。")
	}
}
