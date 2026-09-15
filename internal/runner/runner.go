package runner

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	"devhub/internal/logfmt"
	"devhub/internal/model"
	"devhub/internal/store"
)

type running struct {
	cmd      *exec.Cmd
	kill     func() error
	done     chan struct{}
	once     sync.Once
	stopping atomic.Bool
}

func (r *running) stop() {
	r.once.Do(func() {
		r.stopping.Store(true)
		_ = r.kill()
	})
}

type Manager struct {
	mu    sync.Mutex
	items map[string]*running
	exits map[string]string
	store *store.Store
}

func New(s *store.Store) *Manager {
	return &Manager{items: map[string]*running{}, exits: map[string]string{}, store: s}
}

func (m *Manager) ConsumeExit(id string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.exits[id]
	delete(m.exits, id)
	return msg
}

type writer struct {
	store *store.Store
	id    string
}

func (w writer) Write(p []byte) (int, error) { return len(p), w.store.Append(w.id, logfmt.Bytes(p)) }
func (m *Manager) Has(id string) bool        { m.mu.Lock(); defer m.mu.Unlock(); return m.items[id] != nil }
func PortOpen(port int) bool {
	for _, host := range []string{"127.0.0.1", "::1"} {
		c, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), 150*time.Millisecond)
		if err == nil {
			c.Close()
			return true
		}
	}
	return false
}
func (m *Manager) Start(s model.Service) (int32, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items[s.ID] != nil {
		return 0, 0, errors.New("服务已经启动")
	}
	if PortOpen(s.Port) {
		return 0, 0, errors.New("端口已被占用，请先停止对应服务或更改端口")
	}
	if s.Command == "" {
		return 0, 0, errors.New("请先设置启动命令")
	}
	c := command(s.Command)
	c.Dir = s.Directory
	c.Stdout = writer{m.store, s.ID}
	c.Stderr = c.Stdout
	if len(s.Env) > 0 {
		env := os.Environ()
		for _, item := range model.NormalizeEnv(s.Env) {
			env = append(env, item.Name+"="+item.Value)
		}
		c.Env = env
	}
	if err := c.Start(); err != nil {
		return 0, 0, err
	}
	kill, err := contain(c)
	if err != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
		return 0, 0, fmt.Errorf("无法安全管理进程树: %w", err)
	}
	r := &running{cmd: c, kill: kill, done: make(chan struct{})}
	m.items[s.ID] = r
	pid := int32(c.Process.Pid)
	p, _ := process.NewProcess(pid)
	var birth int64
	if p != nil {
		birth, _ = p.CreateTime()
	}
	_ = m.store.Append(s.ID, fmt.Sprintf("[DevHub] 已启动 PID %d\n", pid))
	go func(id string) {
		err := c.Wait()
		stoppedByUser := r.stopping.Load()
		r.stop()
		msg := "[DevHub] 进程已退出\n"
		if err != nil {
			msg = fmt.Sprintf("[DevHub] 进程退出: %v\n", err)
		}
		_ = m.store.Append(id, msg)
		m.mu.Lock()
		if !stoppedByUser {
			detail := "进程已退出"
			if err != nil {
				detail = fmt.Sprintf("进程已退出：%v", err)
			}
			m.exits[id] = detail
		}
		delete(m.items, id)
		close(r.done)
		m.mu.Unlock()
	}(s.ID)
	return pid, birth, nil
}
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	r := m.items[id]
	m.mu.Unlock()
	if r == nil {
		return errors.New("服务不由当前 DevHub 启动")
	}
	r.stop()
	select {
	case <-r.done:
		return nil
	case <-time.After(8 * time.Second):
		return errors.New("等待进程退出超时")
	}
}
func (m *Manager) Close() {
	m.mu.Lock()
	all := make([]*running, 0, len(m.items))
	for _, r := range m.items {
		all = append(all, r)
	}
	m.mu.Unlock()
	for _, r := range all {
		r.stop()
	}
	for _, r := range all {
		select {
		case <-r.done:
		case <-time.After(8 * time.Second):
		}
	}
}
