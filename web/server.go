package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Server 是本地 web dashboard 的 HTTP 服务。默认绑 127.0.0.1(仅本机),带随机 token 防未授权访问。
// 可由调用方指定绑定地址(如 0.0.0.0)对外暴露 —— 见 Listen 的安全说明。
// 输入回注通过 OnInput/OnReview 回调解耦,由 tui/run.go 注入(内部调 program.Send)。
type Server struct {
	hub   *Hub
	token string
	ln    net.Listener
	srv   *http.Server

	// 回调:浏览器提交输入 / review 确认时触发。由调用方注入。
	OnInput  func(text string)
	OnReview func(approve bool)

	// OnListFiles 返回工作区文件相对路径列表,供前端 @ 文件选择器使用。由调用方注入
	// (web 包不能依赖 tui —— tui 已依赖 web,会成环;遍历逻辑在 tui 侧,经回调注入)。
	OnListFiles func() []string

	// 控制类回调:浏览器点按钮 → 注入回 TUI(同样经 program.Send,走相同 Update 逻辑)。
	OnNewSession     func()             // 新建会话(/new)
	OnSwitchSession  func(id string)    // 切换会话(/sessions 点击)
	OnRenameSession  func(id, title string) // 重命名会话
	OnDeleteSession  func(id string)    // 删除会话
	OnSetModel       func(role string) // 路由 auto/flash/pro
	OnSetMode        func(mode string) // 权限模式 plan/auto/review
	OnSetSandbox     func(mode string) // 沙箱 off/native/docker
	OnSetWorkingMode func(mode string) // 工作模式 karpathy/openspec/superpowers
	OnSetLang        func(lang string) // 界面语言 zh/en
}

// NewServer 创建 Server,token 先随机兜底(可被 SetToken 覆盖成 session 固定令牌)。
func NewServer(hub *Hub) *Server {
	return &Server{hub: hub, token: randomToken()}
}

// SetToken 用一个固定令牌覆盖随机令牌(空串忽略)。须在 Listen 前调用,返回的 URL 才带该令牌。
// 给 tui/run.go 注入 session 级固定 token 用,保证同一 workspace 的访问 URL 跨重启稳定。
func (s *Server) SetToken(t string) {
	if t != "" {
		s.token = t
	}
}

// Listen 在 host:port 上监听(host 空 => 127.0.0.1,仅本机;port=0 => 随机空闲端口),
// 返回带 token 的访问 URL。
//
// 安全:该面板可向 agent 注入输入、批准工具调用、关沙箱 —— 等价于在本机执行代码,且为明文 HTTP、
// 令牌在 URL 里。host 设成 0.0.0.0 或某网卡 IP 会把它暴露到局域网/公网,仅应在可信网络这么做。
// 默认 127.0.0.1 不对外。
func (s *Server) Listen(host string, port int) (string, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return "", err
	}
	s.ln = ln
	actual := ln.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("http://%s:%d/?t=%s", displayHost(host), actual, s.token), nil
}

// displayHost 把绑定地址转成给用户点击的 URL host:通配地址(0.0.0.0 / ::)解析成本机第一个
// 非回环 IPv4(便于局域网设备直接访问),其余原样返回。
func displayHost(bind string) string {
	if bind == "0.0.0.0" || bind == "::" {
		return localIP()
	}
	return bind
}

// localIP 返回本机第一个非回环 IPv4;找不到则回退 127.0.0.1。
// best-effort:多网卡(docker 网桥 / VPN / 多 NIC)时取到的不一定是用户期望的那张,仅用于 URL 回显,
// 不影响实际监听(已绑在所有接口)。
func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

// Relisten 关掉当前监听,在新的 host:port 重新监听并起服务,返回新访问 URL。
// token / hub 不变(已打开的浏览器刷新即可继续用)。供 /web-config 改完配置后热生效,免重启。
func (s *Server) Relisten(host string, port int) (string, error) {
	if s.srv != nil {
		_ = s.srv.Close() // 关旧 http.Server,旧 Serve goroutine 随之返回
		s.srv = nil
	} else if s.ln != nil {
		_ = s.ln.Close()
	}
	url, err := s.Listen(host, port)
	if err != nil {
		return "", err
	}
	go func() { _ = s.Serve() }()
	return url, nil
}

// Serve 启动 HTTP 服务(阻塞)。通常放进 goroutine。Close 后返回。
func (s *Server) Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/input", s.handleInput)
	mux.HandleFunc("/api/review", s.handleReview)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/new", s.handleNew)
	mux.HandleFunc("/api/switch", s.handleSwitch)
	mux.HandleFunc("/api/session-rename", s.handleSessionRename)
	mux.HandleFunc("/api/session-delete", s.handleSessionDelete)
	mux.HandleFunc("/api/model", s.handleModel)
	mux.HandleFunc("/api/mode", s.handleMode)
	mux.HandleFunc("/api/sandbox", s.handleSandbox)
	mux.HandleFunc("/api/workingmode", s.handleWorkingMode)
	mux.HandleFunc("/api/lang", s.handleLang)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	err := s.srv.Serve(s.ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Close 关闭服务。
func (s *Server) Close() {
	if s.srv != nil {
		_ = s.srv.Close()
	} else if s.ln != nil {
		_ = s.ln.Close()
	}
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "deepx" // 极端兜底,实际 crypto/rand 不会失败
	}
	return hex.EncodeToString(b)
}

// authed 校验请求是否带正确 token(query ?t= 或 cookie)。
func (s *Server) authed(r *http.Request) bool {
	if r.URL.Query().Get("t") == s.token {
		return true
	}
	if c, err := r.Cookie("deepx_token"); err == nil && c.Value == s.token {
		return true
	}
	return false
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// 首次带 token 访问 → 种 cookie,后续 API 请求免带 ?t=。
	if r.URL.Query().Get("t") == s.token {
		http.SetCookie(w, &http.Cookie{
			Name: "deepx_token", Value: s.token, Path: "/", HttpOnly: true,
		})
	}
	// 禁止浏览器缓存内嵌静态资源 —— go:embed 文件 modtime 为零值,否则浏览器会一直用旧的
	// app.js/css,deepx 升级后前端不更新(用户曾遇到改了算法但页面还是旧逻辑)。
	w.Header().Set("Cache-Control", "no-store")
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		http.Error(w, "embed error", http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(sub)).ServeHTTP(w, r)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, snap, unsub := s.hub.Subscribe()
	defer unsub()

	writeSSE(w, "snapshot", snap)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return // hub 关闭了该客户端(慢消费者),浏览器会自动重连
			}
			writeSSE(w, "delta", ev)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

func (s *Server) handleInput(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if s.OnInput != nil && body.Text != "" {
		s.OnInput(body.Text)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Approve bool `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if s.OnReview != nil {
		s.OnReview(body.Approve)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.hub.SnapshotCopy())
}

// handleFiles 返回工作区文件相对路径列表(JSON 数组),供前端 @ 文件选择器过滤。
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var files []string
	if s.OnListFiles != nil {
		files = s.OnListFiles()
	}
	if files == nil {
		files = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

// postField 是控制类 POST 端点的公共骨架:校验 auth + method,解析 body 里的 field 字段,
// 调回调。field 为空字符串时表示无 body(如新建会话),cb 直接调。
func (s *Server) postField(w http.ResponseWriter, r *http.Request, field string, cb func(string)) {
	if !s.authed(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	val := ""
	if field != "" {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		val = body[field]
	}
	if cb != nil {
		cb(val)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNew(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "", func(string) {
		if s.OnNewSession != nil {
			s.OnNewSession()
		}
	})
}

func (s *Server) handleSwitch(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "id", func(id string) {
		if s.OnSwitchSession != nil && id != "" {
			s.OnSwitchSession(id)
		}
	})
}

func (s *Server) handleSessionRename(w http.ResponseWriter, r *http.Request) {
	if !s.authed(r) || r.Method != http.MethodPost {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if s.OnRenameSession != nil && body.ID != "" {
		s.OnRenameSession(body.ID, body.Title)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "id", func(id string) {
		if s.OnDeleteSession != nil && id != "" {
			s.OnDeleteSession(id)
		}
	})
}

func (s *Server) handleLang(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "lang", func(lang string) {
		if s.OnSetLang != nil && lang != "" {
			s.OnSetLang(lang)
		}
	})
}

func (s *Server) handleModel(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "role", func(role string) {
		if s.OnSetModel != nil && role != "" {
			s.OnSetModel(role)
		}
	})
}

func (s *Server) handleMode(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "mode", func(m string) {
		if s.OnSetMode != nil && m != "" {
			s.OnSetMode(m)
		}
	})
}

func (s *Server) handleSandbox(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "mode", func(m string) {
		if s.OnSetSandbox != nil && m != "" {
			s.OnSetSandbox(m)
		}
	})
}

func (s *Server) handleWorkingMode(w http.ResponseWriter, r *http.Request) {
	s.postField(w, r, "mode", func(m string) {
		if s.OnSetWorkingMode != nil && m != "" {
			s.OnSetWorkingMode(m)
		}
	})
}
