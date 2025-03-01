package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/olekukonko/tablewriter"
)

// 复杂度告警阈值，可根据需要自行调整
const DefaultComplexityThreshold = 15

// 并发处理的工作池大小
const DefaultWorkerCount = 8

// FileStats 文件统计信息
type FileStats struct {
	Path       string
	Module     string // 所属模块
	Package    string // 所属包
	Language   string
	Lines      int
	CodeLines  int
	Comments   int
	BlankLines int

	Imports   []string
	Functions []FunctionInfo
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name       string
	Lines      int
	Complexity int    // 圈复杂度
	Receiver   string // 接收者类型（如果是方法）
	StartLine  int    // 函数起始行号，便于快速定位
}

// ProjectStats 项目统计信息
type ProjectStats struct {
	RootDir       string
	ModuleName    string
	Files         map[string]*FileStats
	Packages      map[string][]string // 包名 -> 文件列表
	Languages     map[string]int
	Dependencies  map[string][]string // 包级依赖
	FuncCalls     map[string][]string // 函数调用关系（如需可扩展收集）
	TotalLines    int
	TotalCode     int
	TotalComments int
	TotalBlank    int

	// 复杂度整体分析
	MaxComplexity   int
	AvgComplexity   float64
	TotalFunctions  int
	OverThresholdFn []FunctionInfo // 超过阈值的函数
}

// main 入口
func main() {
	var dir string
	flag.StringVar(&dir, "dir", "", "项目根目录路径（默认自动查找）")
	flag.Parse()

	// 如果未指定目录，查找最近的包含 go.mod 的目录
	if dir == "" {
		dir = findProjectRoot()
		if dir == "" {
			fmt.Println("Error: 未找到项目根目录（包含 go.mod 的目录）")
			return
		}
	}

	// 初始化项目统计
	stats := &ProjectStats{
		RootDir:      dir,
		Files:        make(map[string]*FileStats),
		Packages:     make(map[string][]string),
		Languages:    make(map[string]int),
		Dependencies: make(map[string][]string),
		FuncCalls:    make(map[string][]string),
	}

	// 读取 go.mod 获取模块名
	loadModuleName(stats)

	// 收集所有待处理文件
	fileList, err := collectFiles(stats.RootDir)
	if err != nil {
		fmt.Printf("Error collecting files: %v\n", err)
		return
	}

	// 并发处理文件
	analyzeFilesConcurrently(stats, fileList, DefaultWorkerCount)

	// 打印报告和分析结果
	printProjectSummary(stats)
	printModuleStructure(stats)
	printLanguageStats(stats)
	printPackageStats(stats)
	printLargestFiles(stats)
	printDependencyGraph(stats)
	printCouplingMatrix(stats)

	// 函数级别的综合分析
	printFunctionComplexityAnalysis(stats)
	printTopNComplexFunctions(stats, 15)
	printTopNFunctionByLines(stats, 15)
}

//========================================
// 收集文件并构建初步结构
//========================================

// findProjectRoot 查找包含 go.mod 的最近父目录
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadModuleName 从 go.mod 读取模块名
func loadModuleName(stats *ProjectStats) {
	modBytes, err := ioutil.ReadFile(filepath.Join(stats.RootDir, "go.mod"))
	if err != nil {
		return
	}
	lines := strings.Split(string(modBytes), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "module ") {
		stats.ModuleName = strings.TrimSpace(strings.TrimPrefix(lines[0], "module "))
	}
}

// collectFiles 遍历目录收集所有需要处理的文件
func collectFiles(root string) ([]string, error) {
	var fileList []string

	// 获取当前执行文件的绝对路径
	selfPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %v", err)
	}

	// 转换为规范化的绝对路径
	selfPath, err = filepath.EvalSymlinks(selfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to eval symlinks: %v", err)
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对于根目录的路径
		relPath, err := filepath.Rel(root, path)
		if err == nil {
			// 跳过 tools/codestat 目录
			if strings.HasPrefix(relPath, "tools/codestat") || strings.HasPrefix(relPath, "tools\\codestat") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 跳过自身文件
		absPath, err := filepath.Abs(path)
		if err == nil {
			absPath, err = filepath.EvalSymlinks(absPath)
			if err == nil && absPath == selfPath {
				return nil
			}
		}

		// 跳过隐藏文件和目录
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// 跳过编译产物(.exe)
		if strings.ToLower(filepath.Ext(info.Name())) == ".exe" {
			return nil
		}
		// 跳过 vendor、node_modules
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "node_modules") {
			return filepath.SkipDir
		}

		// 只处理文件
		if !info.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList, err
}

//========================================
// 并发分析文件
//========================================

// analyzeFilesConcurrently 并发处理所有文件
func analyzeFilesConcurrently(stats *ProjectStats, files []string, workerCount int) {
	jobs := make(chan string, len(files))
	results := make(chan *FileStats, len(files))

	var wg sync.WaitGroup

	// 启动 worker
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				fileStats, err := analyzeFile(stats.RootDir, stats.ModuleName, path)
				if err != nil {
					fmt.Printf("Error analyzing file %s: %v\n", path, err)
					continue
				}
				if fileStats != nil {
					results <- fileStats
				}
			}
		}()
	}

	// 分配任务
	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	// 等待 worker 完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 汇总结果
	for res := range results {
		// 存储 FileStats
		stats.Files[res.Path] = res
		stats.Languages[res.Language]++

		stats.TotalLines += res.Lines
		stats.TotalCode += res.CodeLines
		stats.TotalComments += res.Comments
		stats.TotalBlank += res.BlankLines

		// 记录包信息
		if res.Package != "" {
			stats.Packages[res.Package] = append(stats.Packages[res.Package], res.Path)
		}

		// 记录依赖关系（仅项目内部）
		if len(res.Imports) > 0 {
			pkgImports := filterProjectImports(stats.ModuleName, res.Imports)
			if len(pkgImports) > 0 && res.Package != "" {
				// 每个包仅记录一次，去重可自行处理
				stats.Dependencies[res.Package] = appendUnique(
					stats.Dependencies[res.Package],
					pkgImports...,
				)
			}
		}

		// 统计全局函数复杂度
		for _, fn := range res.Functions {
			stats.TotalFunctions++
			if fn.Complexity > stats.MaxComplexity {
				stats.MaxComplexity = fn.Complexity
			}
			// 收集超阈值函数
			if fn.Complexity >= DefaultComplexityThreshold {
				stats.OverThresholdFn = append(stats.OverThresholdFn, fn)
			}
			stats.AvgComplexity += float64(fn.Complexity)
		}
	}
	if stats.TotalFunctions > 0 {
		stats.AvgComplexity /= float64(stats.TotalFunctions)
	}
}

//========================================
// 文件与函数的具体分析逻辑
//========================================

// analyzeFile 分析单个文件的行数、注释、函数等信息
func analyzeFile(rootDir, moduleName, path string) (*FileStats, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(rootDir, path)
	stats := &FileStats{
		Path:     relPath,
		Language: getFileLanguage(path),
		Module:   moduleName,
	}

	// 统计行数
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inComment := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		stats.Lines++

		if line == "" {
			stats.BlankLines++
			continue
		}

		if strings.HasPrefix(line, "/*") {
			inComment = true
			stats.Comments++
			continue
		}
		if inComment {
			stats.Comments++
			if strings.HasSuffix(line, "*/") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(line, "//") {
			stats.Comments++
			continue
		}
		stats.CodeLines++
	}

	// 如果是 Go 文件，解析 AST 获取更详细的信息
	if stats.Language == "Go" {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
		if err == nil {
			// 获取包名
			stats.Package = f.Name.Name

			// 收集 import
			for _, imp := range f.Imports {
				if imp.Path != nil {
					importPath := strings.Trim(imp.Path.Value, "\"")
					stats.Imports = append(stats.Imports, importPath)
				}
			}

			// 收集函数信息
			ast.Inspect(f, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					info := FunctionInfo{
						Name:       fn.Name.Name,
						Lines:      countFunctionLines(fn, fset),
						Complexity: calculateComplexity(fn),
						StartLine:  fset.Position(fn.Pos()).Line,
					}
					if fn.Recv != nil && len(fn.Recv.List) > 0 {
						info.Receiver = getTypeString(fn.Recv.List[0].Type)
					}
					stats.Functions = append(stats.Functions, info)
				}
				return true
			})
		}
	}
	return stats, nil
}

// countFunctionLines 计算函数的行数
func countFunctionLines(fn *ast.FuncDecl, fset *token.FileSet) int {
	if fn.Body == nil {
		return 0
	}
	start := fset.Position(fn.Pos()).Line
	end := fset.Position(fn.End()).Line
	return end - start + 1
}

// calculateComplexity 计算函数圈复杂度（简单地统计分支节点）
func calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // 基础复杂度
	ast.Inspect(fn, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause,
			*ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			if node.Op.String() == "&&" || node.Op.String() == "||" {
				complexity++
			}
		}
		return true
	})
	return complexity
}

// getTypeString 获取函数接收者类型的字符串
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// filterProjectImports 只筛选属于当前模块内部的 import
func filterProjectImports(moduleName string, imports []string) []string {
	var filtered []string
	for _, imp := range imports {
		if strings.HasPrefix(imp, moduleName) {
			filtered = append(filtered, strings.TrimPrefix(imp, moduleName+"/"))
		}
	}
	return filtered
}

// getFileLanguage 根据扩展名判断语言类型
func getFileLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "Go"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	case ".c":
		return "C"
	case ".cpp":
		return "C++"
	case ".h":
		return "Header"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	default:
		return "Other"
	}
}

// appendUnique 将 items 追加到 slice 中，并去重
func appendUnique(slice []string, items ...string) []string {
	m := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		m[s] = struct{}{}
	}
	for _, i := range items {
		if _, exists := m[i]; !exists {
			slice = append(slice, i)
			m[i] = struct{}{}
		}
	}
	return slice
}

//========================================
// 各种统计输出
//========================================

// printProjectSummary 总体统计信息
func printProjectSummary(stats *ProjectStats) {
	fmt.Printf("\n=== Project Summary (%s) ===\n", stats.ModuleName)
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Metric", "Value"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	table.Append([]string{"Root Directory", stats.RootDir})
	table.Append([]string{"Total Files", fmt.Sprintf("%d", len(stats.Files))})
	table.Append([]string{"Total Packages", fmt.Sprintf("%d", len(stats.Packages))})
	table.Append([]string{"Total Lines", fmt.Sprintf("%d", stats.TotalLines)})
	table.Append([]string{"Code Lines", fmt.Sprintf("%d", stats.TotalCode)})
	table.Append([]string{"Comment Lines", fmt.Sprintf("%d", stats.TotalComments)})
	table.Append([]string{"Blank Lines", fmt.Sprintf("%d", stats.TotalBlank)})
	table.Append([]string{"Languages", fmt.Sprintf("%d", len(stats.Languages))})

	table.Render()
}

// printModuleStructure 打印模块/包目录树
func printModuleStructure(stats *ProjectStats) {
	fmt.Println("\n=== Module Structure ===")

	// 按包路径排序
	packages := make([]string, 0, len(stats.Packages))
	for pkg := range stats.Packages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	// 构建目录树
	root := make(map[string]interface{})
	for _, pkg := range packages {
		parts := strings.Split(pkg, "/")
		current := root
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = stats.Packages[pkg]
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				next, ok := current[part].(map[string]interface{})
				if !ok {
					next = make(map[string]interface{})
					current[part] = next
				}
				current = next
			}
		}
	}

	// 打印目录树
	printModuleTree(root, "")
}

// printModuleTree 递归打印目录树
func printModuleTree(tree map[string]interface{}, prefix string) {
	items := make([]string, 0, len(tree))
	for item := range tree {
		items = append(items, item)
	}
	sort.Strings(items)

	for i, item := range items {
		isLast := i == len(items)-1
		marker := "├── "
		if isLast {
			marker = "└── "
		}
		fmt.Printf("%s%s%s\n", prefix, marker, item)

		if files, ok := tree[item].([]string); ok {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			for j, file := range files {
				fileMarker := "├── "
				if j == len(files)-1 {
					fileMarker = "└── "
				}
				fmt.Printf("%s%s%s\n", newPrefix, fileMarker, filepath.Base(file))
			}
		} else if subtree, ok := tree[item].(map[string]interface{}); ok {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			printModuleTree(subtree, newPrefix)
		}
	}
}

// printLanguageStats 按语言的统计
func printLanguageStats(stats *ProjectStats) {
	fmt.Println("\n=== Language Statistics ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Language", "Files", "Lines", "Code%"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	type langStat struct {
		files int
		lines int
		code  int
	}
	langStats := make(map[string]*langStat)
	for _, f := range stats.Files {
		if _, ok := langStats[f.Language]; !ok {
			langStats[f.Language] = &langStat{}
		}
		ls := langStats[f.Language]
		ls.files++
		ls.lines += f.Lines
		ls.code += f.CodeLines
	}

	// 按文件数排序
	languages := make([]string, 0, len(langStats))
	for lang := range langStats {
		languages = append(languages, lang)
	}
	sort.Slice(languages, func(i, j int) bool {
		return langStats[languages[i]].files > langStats[languages[j]].files
	})

	for _, lang := range languages {
		ls := langStats[lang]
		codePercent := 0.0
		if ls.lines > 0 {
			codePercent = float64(ls.code) / float64(ls.lines) * 100
		}
		table.Append([]string{
			lang,
			fmt.Sprintf("%d", ls.files),
			fmt.Sprintf("%d", ls.lines),
			fmt.Sprintf("%.1f%%", codePercent),
		})
	}
	table.Render()
}

// printPackageStats 包层面的统计
func printPackageStats(stats *ProjectStats) {
	fmt.Println("\n=== Package Statistics ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Package", "Files", "Functions", "Complexity", "Dependencies"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	type pkgStat struct {
		files      int
		funcs      int
		complexity int
		deps       int
	}

	pkgStats := make(map[string]*pkgStat)
	for pkg, files := range stats.Packages {
		ps := &pkgStat{files: len(files)}
		for _, file := range files {
			if f, ok := stats.Files[file]; ok {
				ps.funcs += len(f.Functions)
				for _, fn := range f.Functions {
					ps.complexity += fn.Complexity
				}
			}
		}
		ps.deps = len(stats.Dependencies[pkg])
		pkgStats[pkg] = ps
	}

	packages := make([]string, 0, len(pkgStats))
	for pkg := range pkgStats {
		packages = append(packages, pkg)
	}
	sort.Slice(packages, func(i, j int) bool {
		return pkgStats[packages[i]].files > pkgStats[packages[j]].files
	})

	for _, pkg := range packages {
		ps := pkgStats[pkg]
		avgComplexity := 0.0
		if ps.funcs > 0 {
			avgComplexity = float64(ps.complexity) / float64(ps.funcs)
		}
		table.Append([]string{
			pkg,
			fmt.Sprintf("%d", ps.files),
			fmt.Sprintf("%d", ps.funcs),
			fmt.Sprintf("%.1f", avgComplexity),
			fmt.Sprintf("%d", ps.deps),
		})
	}
	table.Render()
}

// printLargestFiles 输出行数最多的前 15 个文件
func printLargestFiles(stats *ProjectStats) {
	fmt.Println("\n=== Largest Files ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"File", "Lines", "Functions", "Avg Complexity", "Package"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	files := make([]*FileStats, 0, len(stats.Files))
	for _, f := range stats.Files {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Lines > files[j].Lines
	})

	for i := 0; i < len(files) && i < 15; i++ {
		f := files[i]
		totalComplexity := 0
		for _, fn := range f.Functions {
			totalComplexity += fn.Complexity
		}
		avgComplexity := 0.0
		if len(f.Functions) > 0 {
			avgComplexity = float64(totalComplexity) / float64(len(f.Functions))
		}
		table.Append([]string{
			f.Path,
			fmt.Sprintf("%d", f.Lines),
			fmt.Sprintf("%d", len(f.Functions)),
			fmt.Sprintf("%.1f", avgComplexity),
			f.Package,
		})
	}
	table.Render()
}

// printDependencyGraph 打印包之间的依赖关系
func printDependencyGraph(stats *ProjectStats) {
	fmt.Println("\n=== Dependency Graph ===")

	packages := make([]string, 0, len(stats.Packages))
	for pkg := range stats.Packages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	visited := make(map[string]bool)
	pathVisited := make(map[string]bool)

	for _, pkg := range packages {
		if !visited[pkg] {
			printPackageDependencyTree(stats, pkg, "", visited, pathVisited)
		}
	}
}

// printPackageDependencyTree 递归打印依赖树
func printPackageDependencyTree(stats *ProjectStats, pkg, prefix string, visited, pathVisited map[string]bool) {
	if pathVisited[pkg] {
		fmt.Printf("%s%s (circular dependency!)\n", prefix, pkg)
		return
	}

	fmt.Printf("%s%s\n", prefix, pkg)
	visited[pkg] = true
	pathVisited[pkg] = true

	deps := stats.Dependencies[pkg]
	sort.Strings(deps)

	for i, dep := range deps {
		isLast := i == len(deps)-1
		newPrefix := prefix
		if isLast {
			newPrefix += "└── "
		} else {
			newPrefix += "├── "
		}

		if !pathVisited[dep] {
			printPackageDependencyTree(stats, dep, newPrefix, visited, pathVisited)
		} else {
			fmt.Printf("%s%s\n", newPrefix, dep)
		}
	}

	delete(pathVisited, pkg)
}

// printCouplingMatrix 打印包间耦合矩阵
func printCouplingMatrix(stats *ProjectStats) {
	fmt.Println("\n=== Package Coupling Matrix ===")

	packages := make([]string, 0, len(stats.Packages))
	for pkg := range stats.Packages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	matrix := make([][]string, len(packages)+1)
	for i := range matrix {
		matrix[i] = make([]string, len(packages)+1)
	}

	// 表头
	matrix[0][0] = "Package"
	for i, pkg := range packages {
		name := filepath.Base(pkg)
		matrix[0][i+1] = name
		matrix[i+1][0] = name
	}

	// 填充矩阵
	for i, pkg1 := range packages {
		for j, pkg2 := range packages {
			if i == j {
				matrix[i+1][j+1] = "×"
				continue
			}
			strength := 0
			for _, dep := range stats.Dependencies[pkg1] {
				if dep == pkg2 {
					strength++
				}
			}
			if strength > 0 {
				matrix[i+1][j+1] = fmt.Sprintf("%d", strength)
			} else {
				matrix[i+1][j+1] = " "
			}
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(matrix[0])
	table.SetBorder(false)
	table.SetColumnSeparator("|")
	for i := 1; i < len(matrix); i++ {
		table.Append(matrix[i])
	}
	table.Render()
}

//========================================
// 函数级别的复杂度报告
//========================================

// printFunctionComplexityAnalysis 打印全局函数复杂度概览
func printFunctionComplexityAnalysis(stats *ProjectStats) {
	fmt.Println("\n=== Function Complexity Analysis ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Total Func", "Max Complexity", "Avg Complexity", "Threshold", "Over Threshold"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	table.Append([]string{
		fmt.Sprintf("%d", stats.TotalFunctions),
		fmt.Sprintf("%d", stats.MaxComplexity),
		fmt.Sprintf("%.2f", stats.AvgComplexity),
		fmt.Sprintf("%d", DefaultComplexityThreshold),
		fmt.Sprintf("%d", len(stats.OverThresholdFn)),
	})
	table.Render()

	// 对超过阈值的函数做单独提醒
	if len(stats.OverThresholdFn) > 0 {
		fmt.Println("函数复杂度超过阈值提醒:")
		for _, fn := range stats.OverThresholdFn {
			fmt.Printf("- %s (Complexity=%d)\n", fn.Name, fn.Complexity)
		}
	}
}

// printTopNComplexFunctions 输出复杂度最高的 N 个函数
func printTopNComplexFunctions(stats *ProjectStats, topN int) {
	fmt.Printf("\n=== Top %d Complex Functions ===\n", topN)
	var allFns []struct {
		File string
		Pkg  string
		Func FunctionInfo
	}
	for path, fs := range stats.Files {
		for _, fn := range fs.Functions {
			allFns = append(allFns, struct {
				File string
				Pkg  string
				Func FunctionInfo
			}{
				File: path,
				Pkg:  fs.Package,
				Func: fn,
			})
		}
	}
	sort.Slice(allFns, func(i, j int) bool {
		return allFns[i].Func.Complexity > allFns[j].Func.Complexity
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Function", "Complexity", "File", "Package", "Line"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	for i := 0; i < len(allFns) && i < topN; i++ {
		fn := allFns[i]
		table.Append([]string{
			fn.Func.Name,
			fmt.Sprintf("%d", fn.Func.Complexity),
			fn.File,
			fn.Pkg,
			fmt.Sprintf("%d", fn.Func.StartLine),
		})
	}
	table.Render()
}

// printTopNFunctionByLines 输出行数最多的 N 个函数
func printTopNFunctionByLines(stats *ProjectStats, topN int) {
	fmt.Printf("\n=== Top %d Functions by Lines ===\n", topN)
	var allFns []struct {
		File string
		Pkg  string
		Func FunctionInfo
	}
	for path, fs := range stats.Files {
		for _, fn := range fs.Functions {
			allFns = append(allFns, struct {
				File string
				Pkg  string
				Func FunctionInfo
			}{
				File: path,
				Pkg:  fs.Package,
				Func: fn,
			})
		}
	}
	sort.Slice(allFns, func(i, j int) bool {
		return allFns[i].Func.Lines > allFns[j].Func.Lines
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Function", "Lines", "File", "Package", "Line"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	for i := 0; i < len(allFns) && i < topN; i++ {
		fn := allFns[i]
		table.Append([]string{
			fn.Func.Name,
			fmt.Sprintf("%d", fn.Func.Lines),
			fn.File,
			fn.Pkg,
			fmt.Sprintf("%d", fn.Func.StartLine),
		})
	}
	table.Render()
}
