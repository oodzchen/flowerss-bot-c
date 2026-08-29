## 使用

命令：

```
/sub [url] 订阅（url 为可选）
/unsub [url] 取消订阅（url 为可选）
/list 查看当前订阅
/set 设置订阅
/check 检查当前订阅
/setfeedtag [sub id] [tag1] [tag2] 设置订阅标签（最多设置三个Tag，以空格分隔）
/setfeedtitle [sub id] [title] 设置订阅标题（省略title恢复RSS原标题）
/setinterval [interval] [sub id] 设置订阅刷新频率（可设置多个sub id，以空格分隔）
/setlang <lang> 设置全部订阅的翻译语言，或 /setlang <sub id> <lang>
/settz <timezone> 设置全部订阅的时区，或 /settz <sub id> <timezone>
/setpreview <count> 设置订阅正文预览字符数与截取方向（正数前N字，负数后N字），或 /setpreview <sub id> <count>
/activeall 开启所有订阅
/pauseall 暂停所有订阅
/import 导入 OPML 文件
/export 导出 OPML 文件
/unsuball 取消所有订阅
/help 帮助
```

### Channel 订阅使用方法

1. 将 Bot 添加为 Channel 管理员
2. 发送相关命令给 Bot

Channel 订阅支持的命令：

```
/sub @ChannelID [url] 订阅
/unsub @ChannelID [url] 取消订阅
/list @ChannelID 查看当前订阅
/check @ChannelID 检查当前订阅
/unsuball @ChannelID 取消所有订阅
/activeall @ChannelID 开启所有订阅
/setfeedtag @ChannelID [sub id] [tag1] [tag2]  设置订阅标签（最多设置三个Tag，以空格分隔）
/setfeedtitle @ChannelID [sub id] [title] 设置订阅标题（省略title恢复RSS原标题）
/setlang @ChannelID [lang] 或 /setlang @ChannelID [sub id] [lang]
/settz @ChannelID [timezone] 或 /settz @ChannelID [sub id] [timezone]
/setpreview @ChannelID [count] 或 /setpreview @ChannelID [sub id] [count]
/import 导入 OPML 文件
/export @ChannelID 导出 OPML 文件
/pauseall @ChannelID 暂停所有订阅
```

**ChannelID 只有设置为 Public Channel 才有。如果是 Private Channel，可以暂时设置为 Public，订阅完成后改为 Private，不影响 Bot 推送消息。**

例如要给 t.me/debug 频道订阅 [阮一峰的网络日志](http://www.ruanyifeng.com/blog/atom.xml) RSS 更新：

1. 将 Bot 添加到 debug 频道管理员列表中
2. 给 Bot 发送 `/sub @debug http://www.ruanyifeng.com/blog/atom.xml` 命令
