/* deepx web dashboard — Vue 3 global build, 零构建。
   通过 SSE 镜像终端会话,POST 回注输入 / review 确认。 */
const { createApp } = Vue;

marked.setOptions({ breaks: true, gfm: true });

// 跟 TUI 同步的 web UI 文案。lang 由后端快照 / lang 事件同步(读 ~/.deepx/meta.json)。
const I18N = {
  zh: {
    connected: '已连接',
    reconnecting: '重连中…',
    streaming: '生成中…',
    you: '你',
    send: '发送',
    'placeholder.idle': '输入消息,Enter 发送,Shift+Enter 换行',
    'placeholder.streaming': '正在生成…(可继续输入,发送后排队)',
    'review.title': '需要确认',
    'review.approve': '批准',
    'review.reject': '拒绝',
    workspace: '工作区',
    'panel.status': '状态',
    'panel.plan': '规划',
    'panel.tools': '工具调用',
    'label.state': '运行',
    'label.prompt': '输入',
    'label.output': '输出',
    'label.cache': '缓存命中',
    'state.streaming': '生成中',
    'state.idle': '空闲',
  },
  en: {
    connected: 'connected',
    reconnecting: 'reconnecting…',
    streaming: 'streaming…',
    you: 'You',
    send: 'Send',
    'placeholder.idle': 'Type a message — Enter to send, Shift+Enter for newline',
    'placeholder.streaming': 'Generating… (you can keep typing; it queues after send)',
    'review.title': 'Confirmation needed',
    'review.approve': 'Approve',
    'review.reject': 'Reject',
    workspace: 'Workspace',
    'panel.status': 'Status',
    'panel.plan': 'Plan',
    'panel.tools': 'Tool Calls',
    'label.state': 'state',
    'label.prompt': 'prompt',
    'label.output': 'output',
    'label.cache': 'cache hit',
    'state.streaming': 'streaming',
    'state.idle': 'idle',
  },
};

createApp({
  data() {
    return {
      messages: [],
      plan: [],
      toolCalls: [],
      usage: null,
      streaming: false,
      models: { flash: '', pro: '', activeRole: 'flash' },
      workspace: '',
      reviewPending: null,
      input: '',
      connected: false,
      openIdx: -1, // 当前流式 assistant 消息下标
      lang: 'zh',  // 跟 TUI 同步
    };
  },
  computed: {
    // 已流式但还没出 token 时显示打字动画
    thinking() {
      return this.streaming && this.openIdx < 0;
    },
    cacheRate() {
      // 显示格式:百分比 + 原始 (hit/prompt),一眼可核对。
      // 百分比跟 TUI 一致用整数除法截断(不四舍五入)。
      if (!this.usage || !this.usage.promptTokens) return '—';
      const hit = this.usage.cacheHit;
      const prompt = this.usage.promptTokens;
      return Math.floor((hit * 100) / prompt) + '% (' + hit + '/' + prompt + ')';
    },
  },
  methods: {
    t(key) {
      const dict = I18N[this.lang] || I18N.zh;
      return dict[key] || (I18N.zh[key] || key);
    },
    render(text) {
      try { return marked.parse(text || ''); } catch (_) { return text || ''; }
    },
    planIcon(s) {
      return { done: '✓', running: '▶', failed: '✗', blocked: '⏸', pending: ' ' }[s] || ' ';
    },
    toolIcon(s) {
      return { done: '✓', running: '▶', failed: '✗' }[s] || '·';
    },
    // mainArg 从工具 args JSON 里抽一个最具代表性的字段,显示在 summary 行(对齐 TUI 的
    // extractMainArg)。完整 args 仍在展开后的 <pre> 里。解析失败 / 无字段返回空串。
    mainArg(tc) {
      if (!tc.args) return '';
      let a;
      try { a = JSON.parse(tc.args); } catch (_) { return ''; }
      if (!a || typeof a !== 'object') return '';
      let v = '';
      switch (tc.name) {
        case 'Read': case 'Write': case 'Update': case 'List': case 'Tree': case 'OCR':
          v = a.path; break;
        case 'Glob':
          v = a.path && a.path !== '.' ? `${a.pattern} in ${a.path}` : a.pattern; break;
        case 'Grep':
          v = a.path ? `${a.pattern} in ${a.path}` : a.pattern; break;
        case 'Search': v = a.query; break;
        case 'Fetch': v = a.url; break;
        case 'LoadSkill': v = a.name; break;
        case 'Bash': v = a.command; break;
        case 'SwitchModel': v = a.reason; break;
        case 'Memory': v = Array.isArray(a.keywords) ? a.keywords.join(' ') : ''; break;
        case 'UpdatePlanStatus':
          v = a.id && a.status ? `${a.id} → ${a.status}` : (a.id || ''); break;
        case 'CreatePlan': v = Array.isArray(a.plans) ? `${a.plans.length} nodes` : ''; break;
        default: v = a.path || '';
      }
      v = (v == null ? '' : String(v)).replace(/\s+/g, ' ').trim();
      return v.length > 80 ? v.slice(0, 77) + '…' : v;
    },
    scrollDown() {
      this.$nextTick(() => {
        const el = this.$refs.msgList;
        if (el) el.scrollTop = el.scrollHeight;
      });
    },
    onKey(e) {
      if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
        e.preventDefault();
        this.send();
      }
    },
    async send() {
      const text = this.input.trim();
      if (!text) return;
      this.input = '';
      await fetch('/api/input', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text }),
      }).catch(() => {});
    },
    async review(approve) {
      await fetch('/api/review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ approve }),
      }).catch(() => {});
    },

    // 用整份快照重置状态(新连接 / 重连)
    applySnapshot(s) {
      this.messages = s.messages || [];
      this.plan = s.plan || [];
      this.toolCalls = s.toolCalls || [];
      this.usage = s.usage || null;
      this.streaming = !!s.streaming;
      this.models = s.models || this.models;
      this.workspace = s.workspace || '';
      this.reviewPending = s.reviewPending || null;
      if (s.lang) this.lang = s.lang;
      // 推断流式气泡:最后一条是 assistant 且还在 streaming 则继续往里追加
      const last = this.messages.length - 1;
      this.openIdx = (this.streaming && last >= 0 && this.messages[last].role === 'assistant') ? last : -1;
      this.scrollDown();
    },

    // 应用单条增量事件(与后端 hub.apply 同构)
    applyEvent(ev) {
      switch (ev.kind) {
        case 'user_message':
          this.messages.push({ role: 'user', content: ev.text || '' });
          this.plan = [];
          this.toolCalls = [];
          this.usage = null;
          this.reviewPending = null;
          this.streaming = true;
          this.openIdx = -1;
          break;
        case 'token':
          if (this.openIdx < 0) {
            this.messages.push({ role: 'assistant', content: '' });
            this.openIdx = this.messages.length - 1;
          }
          this.messages[this.openIdx].content += ev.text || '';
          break;
        case 'reasoning_token':
          break; // 思考过程不入聊天
        case 'tool_call':
          this.toolCalls.push({ id: ev.id, name: ev.name, args: ev.args || '', status: 'running', output: '' });
          break;
        case 'tool_result': {
          const t = this.toolCalls.find((x) => x.id === ev.id) ||
            [...this.toolCalls].reverse().find((x) => x.name === ev.name && x.status === 'running');
          if (t) {
            t.status = ev.success ? 'done' : 'failed';
            t.output = ev.output || '';
          }
          break;
        }
        case 'model_switch':
          if (ev.role) this.models.activeRole = ev.role;
          break;
        case 'plan':
          this.plan = ev.plan || [];
          break;
        case 'plan_status': {
          const p = this.plan.find((x) => x.id === ev.id);
          if (p) {
            if (ev.status) p.status = ev.status;
            if (ev.summary) p.summary = ev.summary;
          }
          break;
        }
        case 'usage':
          this.usage = ev.usage || null;
          break;
        case 'done':
        case 'error':
          this.streaming = false;
          this.openIdx = -1;
          break;
        case 'review_request':
          this.reviewPending = { name: ev.name, args: ev.args || '' };
          break;
        case 'review_resolved':
          this.reviewPending = null;
          break;
        case 'lang':
          if (ev.text) this.lang = ev.text;
          break;
      }
      this.scrollDown();
    },

    connect() {
      const es = new EventSource('/api/events');
      es.addEventListener('snapshot', (e) => {
        this.connected = true;
        this.applySnapshot(JSON.parse(e.data));
      });
      es.addEventListener('delta', (e) => {
        this.applyEvent(JSON.parse(e.data));
      });
      es.onerror = () => { this.connected = false; };
    },
  },
  mounted() {
    this.connect();
  },
}).mount('#app');
