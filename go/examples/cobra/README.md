# Cobra 学习示例：owl

`owl` 是一个用于学习 Cobra 的 CLI 任务管理器，涵盖了 Cobra 的主要功能点。

## 命令结构

```
owl                         根命令
├── task                    任务管理（命令组）
│   ├── task add <name>     添加任务
│   ├── task list           列出任务（别名 ls / l）
│   ├── task done <id...>   标记完成
│   └── task rm <id...>     删除任务（已废弃）
├── server                  服务器管理（命令组）
│   ├── server start        启动服务器
│   ├── server stop         停止服务器
│   └── server status       查看状态（隐藏命令）
└── config                  配置管理（命令组）
    ├── config set <k> <v>  设置配置项
    ├── config get <k>      读取配置项
    ├── config list         列出所有配置（别名 ls）
    └── config reset        重置配置（已废弃）
```

---

## 全局 Flags（持久 flags，所有子命令继承）

| Flag | 短名 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--config` | `-c` | string | `~/.owl.yaml` | 配置文件路径 |
| `--verbose` | `-v` | bool | false | 打印调试信息 |
| `--output` | `-o` | string | `text` | 输出格式：text / json / yaml |

PersistentFlags 定义在根命令上，任意层级的子命令都可以使用：

```bash
owl --verbose task list
owl -o json server start
```

---

## 各命令的设计要点

### task add

演示：`MarkFlagRequired`、`MarkFlagsRequiredTogether`、`MinimumNArgs`、`StringSliceVar`、`RunE`、`RegisterFlagCompletionFunc`

```bash
owl task add "Buy milk" --priority high
owl task add Report --priority medium --due 2026-06-10 --tags work,report
owl task add "Call dentist" --priority low --due 2026-06-15 --remind
```

- `--priority` 是必填 flag，不传直接报错，不会进入命令逻辑
- `--remind` 和 `--due` 必须同时出现或同时不出现（`MarkFlagsRequiredTogether`）
- `--tags` 接受逗号分隔的列表，如 `--tags work,report`，也可以多次传入 `--tags work --tags report`
- 任务名称是位置参数，至少需要一个词（`MinimumNArgs(1)`），多个词自动拼接
- 命令通过 `RunE` 返回 error，cobra 统一处理，不需要在命令内部调用 `os.Exit`
- `--priority` 的补全候选值通过 `RegisterFlagCompletionFunc` 注册为 `low / medium / high`

### task list

演示：`Aliases`、`MarkFlagsMutuallyExclusive`、`NoArgs`、`RegisterFlagCompletionFunc`

```bash
owl task list
owl task ls               # 别名，等同于 task list
owl task list --priority low --tag work
owl task list --all       # --all 和 --status 互斥，不能同时使用
```

- `ls` 和 `l` 是 `list` 的别名（`Aliases`）
- `--all` 和 `--priority` 互斥，同时使用报错（`MarkFlagsMutuallyExclusive`）
- 不接受位置参数（`NoArgs`），多传会报错

### task done / task rm

演示：本地 flags、`Deprecated` 字段、`MinimumNArgs`

```bash
owl task done 1 2 3       # 至少一个 ID
owl task done 42 --force  # --force 是 done 的本地 flag，其他命令看不到它
owl task rm 1             # 可以运行，但打印废弃警告
```

- `--force` 只对 `task done` 生效，是本地 flag（`Flags()`），不会传递给子命令
- `task rm` 设置了 `Deprecated` 字段，命令仍然执行，但会打印提示用户改用 `task done`

### server start

演示：`PreRunE`、`PostRunE`、`DurationVar`

```bash
owl server start
owl server start --host localhost:9090 --timeout 10s --daemon
```

- `PreRunE` 在 `RunE` 之前执行，用于校验前置条件（如检查 host 不为空）
- `PostRunE` 在 `RunE` 成功后执行，用于收尾工作（如打印 "server is ready"）
- `--timeout` 是 `time.Duration` 类型，接受 Go 时间字符串：`30s`、`1m`、`500ms`
- 钩子执行顺序：`PersistentPreRunE（root）→ PreRunE → RunE → PostRunE`

### server status

演示：`Hidden` 字段

```bash
owl server --help         # status 不出现在帮助中
owl server status         # 但命令本身可以正常执行
```

- `Hidden: true` 让命令从帮助输出中消失，但仍然可以调用，适合内部调试命令

### config get

演示：`ValidArgs`、`OnlyValidArgs`、`MatchAll`、`ExactArgs`

```bash
owl config get output     # 合法，output 在预设列表中
owl config get unknown    # 报错，unknown 不在预设列表中
owl config set output json  # 恰好需要两个位置参数
```

- `ValidArgs` 声明合法的位置参数候选值，同时用于 shell 补全
- `OnlyValidArgs` 在运行时拒绝不在 `ValidArgs` 中的值
- `MatchAll` 将多个 Args 校验器组合：`ExactArgs(1)` 且 `OnlyValidArgs` 同时满足才通过

### config reset

演示：`Deprecated` 字段（命令级废弃）

```bash
owl config reset          # 打印废弃警告后仍然执行
```

---

## 根命令的设计要点

演示：`AddGroup`、`PersistentPreRunE`、`SilenceErrors`、`SilenceUsage`

- `AddGroup` 注册命令分组，子命令通过 `GroupID` 关联到对应分组，帮助信息中按组显示
- `PersistentPreRunE` 在**所有子命令**运行前执行，常用于加载配置、校验登录态等全局操作
- `SilenceUsage: true`：命令出错时不打印用法，避免错误信息被淹没
- `SilenceErrors: true`：cobra 不自动打印 error，由 `Execute()` 统一处理输出格式

> **注意**：如果子命令自定义了 `PersistentPreRunE`，会**覆盖**父命令的 `PersistentPreRunE`，需要在子命令中手动调用父命令的钩子。

---

## 快速运行

```bash
go run ./cobra/owl/... --help
go run ./cobra/owl/... task add "Buy milk" --priority high
go run ./cobra/owl/... task ls
go run ./cobra/owl/... server start --host localhost:9090 --timeout 5s
go run ./cobra/owl/... config get output
```

---

## 安装 cobra-cli（脚手架工具）

```bash
go install github.com/spf13/cobra-cli@latest

cobra-cli init          # 初始化项目
cobra-cli add server    # 生成子命令文件
```
