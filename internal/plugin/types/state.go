package types

import (
	"fmt"
	"sync/atomic"
)

// PluginState 插件状态
type PluginState int32

const (
	// StateUninitialized 未初始化
	StateUninitialized PluginState = iota
	// StateInitialized 已初始化
	StateInitialized
	// StateRunning 运行中
	StateRunning
	// StateStopped 已停止
	StateStopped
	// StateError 错误状态
	StateError
)

// String 返回状态的字符串表示
func (s PluginState) String() string {
	switch s {
	case StateUninitialized:
		return "Uninitialized"
	case StateInitialized:
		return "Initialized"
	case StateRunning:
		return "Running"
	case StateStopped:
		return "Stopped"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// StateManager 状态管理器
type StateManager struct {
	state     atomic.Int32
	lastError atomic.Value
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	sm := &StateManager{}
	sm.state.Store(int32(StateUninitialized))
	return sm
}

// GetState 获取当前状态
func (sm *StateManager) GetState() PluginState {
	return PluginState(sm.state.Load())
}

// SetState 设置状态
func (sm *StateManager) SetState(state PluginState) error {
	current := sm.GetState()
	if !isValidTransition(current, state) {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("invalid state transition from %v to %v", current, state),
			nil)
	}
	sm.state.Store(int32(state))
	return nil
}

// SetError 设置错误
func (sm *StateManager) SetError(err error) {
	sm.lastError.Store(err)
	sm.state.Store(int32(StateError))
}

// GetError 获取最后的错误
func (sm *StateManager) GetError() error {
	if err := sm.lastError.Load(); err != nil {
		return err.(error)
	}
	return nil
}

// isValidTransition 检查状态转换是否有效
func isValidTransition(from, to PluginState) bool {
	switch from {
	case StateUninitialized:
		return to == StateInitialized
	case StateInitialized:
		return to == StateRunning || to == StateError
	case StateRunning:
		return to == StateStopped || to == StateError
	case StateStopped:
		return to == StateRunning || to == StateUninitialized || to == StateError
	case StateError:
		return to == StateUninitialized
	default:
		return false
	}
}
