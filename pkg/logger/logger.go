package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	// 基础颜色（清新蓝绿色调）
	debugColor   = color.New(color.FgHiBlue)            // 亮蓝色
	infoColor    = color.New(color.FgHiCyan)            // 亮青色
	warningColor = color.New(color.FgHiYellow)          // 亮黄色
	errorColor   = color.New(color.FgHiRed)             // 亮红色
	fatalColor   = color.New(color.FgHiRed, color.Bold) // 亮红色加粗
)

type rule struct {
	pattern string
	color   *color.Color
}

// 关键词
var highlightRules = []rule{
	// 错误相关（红色，仅在真正错误时使用）
	{`(?i)(error|exception|panic)`, color.New(color.FgHiRed)},
	{`(?i)(failed|fail)`, color.New(color.FgRed)},

	// 时间戳（暗青色）
	{`\d{4}/\d{2}/\d{2}`, color.New(color.FgCyan)},
	{`\d{2}:\d{2}:\d{2}(?:\.\d{3})?`, color.New(color.FgCyan)},

	// IP地址（亮蓝色）
	{`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, color.New(color.FgHiBlue)},

	// 端口号（亮青色）
	{`:\d{2,5}(?:\b|$)`, color.New(color.FgHiCyan)},

	// HTTP方法（亮蓝色）
	{`(?i)\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`, color.New(color.FgBlue)},

	// HTTP状态码
	{`\b([345]\d{2})\b`, color.New(color.FgHiRed)}, // 错误状态码
	{`\b(2\d{2})\b`, color.New(color.FgHiGreen)},   // 成功状态码

	// UUID（亮蓝色）
	{`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, color.New(color.FgHiBlue)},

	// JSON/Map键值（青色）
	{`([a-zA-Z_][a-zA-Z0-9_]*=)`, color.New(color.FgHiCyan)},
	{`"[^"]+":`, color.New(color.FgHiCyan)},

	// 布尔值（亮青色）
	{`\b(true|false)\b`, color.New(color.FgHiCyan)},

	// 路径（亮蓝色）
	{`(?i)([a-z]:\\[\\\w\s.-]+)|(/[\w/.-]+\.\w+)`, color.New(color.FgBlue)},

	// 重要关键词（使用更多蓝绿色）
	{`(?i)\b(success|completed|connected|initialized|started)\b`, color.New(color.FgHiCyan)},
	{`(?i)\b(warning|warn|attention)\b`, color.New(color.FgHiYellow)},
	{`(?i)\b(critical|emergency|alert)\b`, color.New(color.FgHiRed)},

	// 方括号内容（亮蓝色）
	{`\[(.*?)\]`, color.New(color.FgBlue)},

	// 插件相关（使用青色）
	{`\[Plugin\]`, color.New(color.FgHiCyan)},
	{`\[GIN\]`, color.New(color.FgHiCyan)},
	{`\[TOTP\]`, color.New(color.FgHiCyan)},
}

// combinedRegex 用于一次性匹配全部规则的大正则
var combinedRegex *regexp.Regexp

// colorMap[i] 表示第 i 个捕获组对应的颜色
var colorMap []*color.Color

// colorWriter 为标准库 logger 的输出目标
type colorWriter struct{}

// syncPool 管理临时的 strings.Builder，以降低内存分配和 GC 压力
var syncPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// Write 实现 io.Writer 接口
func (cw *colorWriter) Write(p []byte) (int, error) {
	return writeWithColor(p)
}

// writeWithColor 核心：生成日志的前缀 (时间、文件、行号 + 级别标识) + 智能高亮处理
func writeWithColor(bytes []byte) (int, error) {
	// 解析调用者信息 (文件名、行号)
	_, file, line, ok := runtime.Caller(5) // 根据你外层包裹的调用栈层级做调整
	if !ok {
		file = "???"
		line = 0
	}
	file = filepath.Base(file)
	now := time.Now().Format("2006/01/02 15:04:05.000")

	// 从池中获取 builder
	sb := syncPool.Get().(*strings.Builder)
	defer syncPool.Put(sb)
	sb.Reset()

	// 原始日志字符串
	msg := string(bytes)
	upperMsg := strings.ToUpper(msg)

	// 根据前缀快速判断日志级别并设置颜色
	var levelColor *color.Color
	var levelTag string
	switch {
	case strings.Contains(upperMsg, "[DEBUG]"):
		levelColor = debugColor
		levelTag = "[DEBUG]"
	case strings.Contains(upperMsg, "[INFO]"):
		levelColor = infoColor
		levelTag = "[INFO]"
	case strings.Contains(upperMsg, "[WARN]"):
		levelColor = warningColor
		levelTag = "[WARN]"
	case strings.Contains(upperMsg, "[ERROR]"):
		levelColor = errorColor
		levelTag = "[ERROR]"
	case strings.Contains(upperMsg, "[FATAL]"):
		levelColor = fatalColor
		levelTag = "[FATAL]"
	default:
		levelColor = infoColor
		levelTag = ""
	}

	// 移除日志级别标记，避免被正则重复匹配
	if levelTag != "" {
		msg = strings.Replace(msg, levelTag, "", 1)
	}
	msg = strings.TrimSpace(msg)

	// 拼接前缀
	prefix := fmt.Sprintf("%s %s:%d", now, file, line)
	sb.WriteString(color.New(color.FgHiBlue).Sprint(prefix))
	sb.WriteByte(' ')

	// 添加日志级别标记（使用对应颜色）
	if levelTag != "" {
		sb.WriteString(levelColor.Sprint(levelTag))
		sb.WriteByte(' ')
	}

	// 高亮日志内容
	highlighted := highlightMessage(msg)
	sb.WriteString(highlighted)
	sb.WriteByte('\n')

	// 将结果输出到 stdout
	_, _ = os.Stdout.WriteString(sb.String())

	return len(bytes), nil
}

// highlightMessage 使用一个大正则一次性找出所有命中规则的子匹配组，再做区间着色
func highlightMessage(msg string) string {
	// 大正则一次性匹配
	matches := combinedRegex.FindAllStringSubmatchIndex(msg, -1)
	if len(matches) == 0 {
		// 没有任何匹配
		return msg
	}

	// 收集所有 (start, end, color)
	type interval struct {
		start int
		end   int
		color *color.Color
	}
	var intervals []interval

	// 每条 match: 其 submatchIndex = [ start_of_full, end_of_full, start_of_group1, end_of_group1, start_of_group2, ... ]
	for _, m := range matches {
		// m[0], m[1] 是整条匹配的开始结束
		// submatch 后续下标按捕获组顺序排列
		subCount := len(m)/2 - 1 // 0号是整条匹配，后面才是各子匹配组

		// 找到哪个子匹配组非 -1，即表示该组匹配到了
		for i := 0; i < subCount && i < len(colorMap); i++ {
			grpStart := m[2+2*i]
			grpEnd := m[2+2*i+1]
			if grpStart >= 0 && grpEnd >= 0 && grpEnd <= len(msg) {
				intervals = append(intervals, interval{
					start: grpStart,
					end:   grpEnd,
					color: colorMap[i],
				})
			}
		}
	}

	// 如果没有匹配到任何捕获组，就直接返回
	if len(intervals) == 0 {
		return msg
	}

	// 对所有区间按开始位置排序
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})

	// 合并区间 + 构建高亮字符串
	var result strings.Builder
	result.Grow(len(msg))

	cur := 0
	for i := 0; i < len(intervals); i++ {
		iv := intervals[i]
		if iv.start < cur {
			// 已被前一个区间覆盖，跳过
			continue
		}
		// 先写正常部分
		if iv.start > cur {
			result.WriteString(msg[cur:iv.start])
		}
		// 写高亮部分
		result.WriteString(iv.color.Sprint(msg[iv.start:iv.end]))
		cur = iv.end
	}
	// 收尾部分
	if cur < len(msg) {
		result.WriteString(msg[cur:])
	}
	return result.String()
}

func init() {
	// 构造单一大正则 (每个规则放到一个独立捕获组里)
	var sb strings.Builder
	sb.Grow(1024)

	colorMap = make([]*color.Color, 0, len(highlightRules))

	sb.WriteByte('(') // 整个要加一个分组，表示一组可选
	for i, r := range highlightRules {
		if i > 0 {
			sb.WriteByte('|')
		}
		// 每条规则都是 (PATTERN)
		sb.WriteString("(")
		sb.WriteString(r.pattern)
		sb.WriteString(")")
		colorMap = append(colorMap, r.color)
	}
	sb.WriteByte(')')
	// 形如：((?i)(?:error|fail|...)|(\d{2}:\d{2}:\d{2})|(...))

	combinedRegex = regexp.MustCompile(sb.String())

	// 替换标准库日志输出
	log.SetOutput(&colorWriter{})
	log.SetFlags(0)
}

func Debug(format string, v ...interface{}) {
	log.Printf("[DEBUG] "+format, v...)
}

func Info(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func Warn(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

func Error(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func Fatal(format string, v ...interface{}) {
	log.Printf("[FATAL] "+format, v...)
	os.Exit(1)
}
