# 插件自动导入机制

本文档介绍了插件系统中的自动导入机制，该机制通过自动扫描插件目录并生成必要的导入代码，简化了插件的管理和维护。

## 工作原理

系统使用以下方式实现插件的自动导入：

1. `auto_import.go`文件中的`go:generate`指令指向`tools/plugin_scanner.go`工具
2. 扫描工具会自动检查`internal/plugin`目录下的所有子目录
3. 系统根据严格的条件识别有效的插件目录：
   - 存在`init.go`文件，或
   - 文件中包含插件注册代码（如`RegisterPlugin`调用）
4. 工具自动生成`import_gen.go`文件，包含所有识别到的插件的导入语句

## 使用方法

有两种方式可以触发插件的自动导入：

### 方法一：使用Go Generate

```bash
go generate ./internal/plugin/...
```

该命令会执行`auto_import.go`文件中的`go:generate`指令，扫描插件目录并生成导入文件。

### 方法二：直接运行扫描工具

```bash
go run tools/plugin_scanner.go
```

这种方式会直接执行扫描工具，效果与使用`go generate`相同。

## 添加新插件

添加新插件的步骤：

1. 在`internal/plugin`目录下创建新的插件子目录
2. 实现必要的插件接口
3. **确保包含`init.go`文件或在文件中使用`types.RegisterPlugin()`注册插件**
4. 运行上述命令之一来更新导入列表
5. 新插件将自动被系统识别和导入

不需要手动编辑任何导入文件，系统会自动处理所有导入关系。

## 插件目录结构

一个标准的插件目录应包含：

- `init.go`：包含`init()`函数，用于注册插件工厂函数，例如：
  ```go
  func init() {
      types.RegisterPlugin("plugin_name", func() types.Plugin {
          return NewPlugin()
      })
  }
  ```
- 插件实现文件：实现`types.Plugin`接口的各种功能

## 插件识别规则

系统使用以下规则来识别有效的插件：

1. 目录中包含名为`init.go`的文件
2. 或者目录中的Go文件包含以下任一模式：
   - `types.RegisterPlugin(`调用
   - `func init() {`函数定义
   - `RegisterPlugin(`调用

