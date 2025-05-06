<p align="right">
  <a href="readme_cn.md">简体中文</a> |
  <a href="readme.md">English</a>
</p>

# IP Radar - 网络IP监控工具

## 项目描述

一个用Go语言编写的网络监控工具，用于检测本地网络接口的IP地址变化并通过电子邮件发送通知。

## 功能特性

- 自动检测本地网络接口的IP地址变化
- 支持IPv4和IPv6地址
- 通过SMTP发送精美的HTML格式邮件通知
- 内置Web配置界面(端口8087)
- 自动创建默认配置文件
- 每10分钟自动检查IP变化

## 安装指南

### 前提条件

- Go 1.20或更高版本
- 有效的SMTP邮箱配置

### 安装步骤

1. 克隆或下载项目代码
2. 确保Go环境已正确配置
3. 运行以下命令安装依赖并构建程序：

   ```
   go mod init ip-radar
   go build
   ```

## 使用方法

1. 首次运行会自动创建`config.json`配置文件
2. 通过浏览器访问 [http://localhost:8087](http://localhost:8087) 配置SMTP邮箱设置
3. 程序会自动运行并监控IP地址变化

## 配置说明

配置文件`config.json`包含以下字段：

```
{
  "email": {
    "from": "发件人邮箱",
    "password": "邮箱密码/授权码",
    "to": "收件人邮箱",
    "smtpHost": "SMTP服务器地址",
    "smtpPort": "SMTP端口"
  }
}
```

## 代码结构

```
main.go - 程序入口文件
config.json - 配置文件(自动生成)
```

## 常见问题

### 为什么检测不到我的IP地址？

程序会跳过以下类型的IP地址：

- 环回地址(127.0.0.1等)
- 链路本地地址
- 私有地址(10.0.0.0/8等)
- 虚拟接口(VMware, VirtualBox等)

### 如何修改检查频率？

修改`main()`函数中的`time.Sleep(10 * time.Minute)`参数

### 为什么收不到邮件通知？

请检查：

1. SMTP配置是否正确
2. 邮箱是否启用了SMTP服务
3. 邮件是否被归类为垃圾邮件

![config](./doc/config.png "Web配置页面")

![email](./doc/email.png "接收邮件示例")

## 许可证
MIT