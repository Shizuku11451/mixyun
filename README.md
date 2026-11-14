<p align="center"><strong>简体中文</strong> · <a href="README.en.md">English</a></p>

<p align="center">
  <img src="brand/icon.png" width="112" height="112" alt="混云的图标：黑色背景上的两个粉色圆环和一个播放按钮" />
</p>

<h1 align="center">混云 Mixyun</h1>

<p align="center"><strong>网易云音乐和 Spotify，放进同一个播放器。</strong></p>

<p align="center">适用于 macOS、Windows 和 Linux 的桌面音乐播放器，中文界面，逐字歌词，可下载无损音乐。</p>

<p align="center">
  <a href="https://github.com/Shizuku11451/mixyun/releases/latest"><img src="https://img.shields.io/github/v/release/Shizuku11451/mixyun" alt="最新版本" /></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey" alt="macOS、Windows 和 Linux" />
  <a href="#许可证"><img src="https://img.shields.io/badge/license-MIT%20or%20GPL--3.0-blue" alt="MIT 或 GPL-3.0" /></a>
</p>

<p align="center">
  <a href="https://github.com/Shizuku11451/mixyun/releases/latest">下载</a> ·
  <a href="CHANGELOG.md">更新日志</a>
</p>

> 混云是 [MixMusic](https://github.com/fzn0x/mixmusic) 的分支。原项目把 YouTube Music 和 Spotify 放在一起；混云把 YouTube Music 换成了网易云音乐，并加上了中文界面和更多面向中文用户的功能。感谢 MixMusic 的作者和贡献者。

## 功能

### 播放

- **网易云音乐 + Spotify**：一个搜索框同时搜两边，歌曲可以混在同一个播放队列和歌单里。
- **音质可选**：在 设置 › 播放 › 音质 里选择，最高可播放无损 FLAC（需要账号有相应权限），也可以用 128 kbit/s 省流量。
- **下载**：网易云音乐的歌曲按原格式（MP3 或 FLAC）下载，没有网络也能播放。
- **漫游**：按标签连续播放；登录网易云音乐后，还能按你喜欢的歌漫游（心动模式），队列播完会自动续上。
- **本地音乐**：播放电脑里的 MP3、FLAC、WAV、M4A、Opus 和 Vorbis 文件。

### 歌词

- **逐字歌词**：网易云音乐有逐字时间轴的歌，会一个字一个字地亮起来。
- **歌词翻译**：优先用网易云音乐里听众写的中文翻译，没有时再用机器翻译。
- **状态栏歌词**：在 macOS 菜单栏或 Windows 任务栏显示正在唱的那一句。
- **日文罗马音**：日文歌词可以显示罗马音。

### 其他

- **搜索联想**：边打字边给出建议，包括歌曲、歌手、专辑和歌单。
- **统计**：播放热力图、常听的语言和曲风，以及每个服务各播了多少首。
- **Discord 状态**：在 Discord 个人资料上显示正在听的歌。
- **主题和插件**：自带多种布局和主题，支持社区主题和插件，所有命令都可以设置快捷键。
- **远程控制和投屏**：在局域网里用另一台电脑控制播放，或投到 Chromecast 等设备上。
- **AI 助手**：支持 MCP 协议的 AI 助手可以查询和控制播放。
- **中文界面**：界面完整翻译成简体中文。

## 安装

到 [Releases 页面](https://github.com/Shizuku11451/mixyun/releases/latest) 下载对应系统的安装包：

| 系统 | 文件 |
| --- | --- |
| macOS 12 及以上（Apple 芯片和 Intel） | `mixmusic-版本号-macos-universal.dmg` |
| Windows 10 / 11 | `mixmusic-版本号-windows-amd64-setup.exe`，或免安装的 `-portable.zip` |
| Debian / Ubuntu | `.deb` |
| Fedora / openSUSE | `.rpm` |
| 任意 Linux | `.AppImage` 或 `.flatpak` |

> 安装包文件名暂时沿用 `mixmusic-` 开头，装好后应用名就是混云（Mixyun）。

**macOS 首次打开**：应用还没有经过 Apple 公证，第一次需要在 系统设置 › 隐私与安全性 里点"仍要打开"，或者在终端运行：

```sh
xattr -dr com.apple.quarantine /Applications/Mixyun.app
```

**Windows 首次打开**：安装包还没有签名，如果 SmartScreen 弹出警告，点"更多信息" › "仍要运行"。

装好后应用会自动检查并安装更新。

## 使用

- **网易云音乐**：在 设置 › 账号 登录后，可以用自己的歌单、喜欢的音乐和心动模式，VIP 账号还能播放更高音质。
- **Spotify**：需要 Spotify Premium，在 设置 › 账号 登录。

## 从源码构建

需要 [Go](https://go.dev/dl/) 1.27.3 或更高版本、[Bun](https://bun.sh) 1.2 或更高版本，以及 Git：

```sh
git clone https://github.com/Shizuku11451/mixyun
cd mixyun
bun install
bun run dev     # 开发模式运行
bun run build   # 构建安装包，输出到 build/
```

Linux 还需要 GTK、PulseAudio、WebKitGTK 等运行库，详见[英文 README](README.en.md)。

## 隐私

混云没有自己的账号体系和服务器，也不做统计和遥测。登录信息保存在系统钥匙串里，只发给对应的音乐服务。

## 许可证

可任选 [MIT](LICENSE-MIT) 或 [GPL-3.0](LICENSE-GPL) 使用，与原项目 MixMusic 相同。
