# PC Multitool

A modern web app and terminal engineering calculator/multitool for **Plane Crazy**.

**Made by Xad0**

[![Latest Release](https://img.shields.io/github/v/release/xadomas2012/Plane-crazy-multitool?label=latest%20release)](https://github.com/xadomas2012/Plane-crazy-multitool/releases/latest)

## Web App

**Use the web app instantly — no installation required.**

🌐 **[Open PC Multitool](https://xad0.github.io/Plane-crazy-multitool/)**

The web app is responsive and works on desktop, tablet, and mobile browsers.

It includes:

* ⚙️ Gear calculator
* 🔩 Crank calculator
* 🛞 Wheel calculator
* 📈 Dyno calculator and graph
* 🧱 Piston calculator
* 📊 Gear reference chart
* 🎨 Multiple themes and accent colors
* 💾 Persistent settings in your browser
* 📱 Mobile-friendly responsive interface
* 🖼️ Dyno graph PNG export

## Features

* ⚙️ Gear angle and offset calculations
* 🔩 Automatic compressor calculation
* 🎛️ Manual compressor adjustment
* 📊 4–20 teeth reference chart
* 🖱️ Mouse support
* 🖥️ Fullscreen terminal UI
* 🎨 Catppuccin Mocha, Nord, Gruvbox and Solid themes
* 🌈 Multiple accent colors
* 🔄 Built-in updater
* 🧮 Designed for quick in-game engineering calculations

## Download

The latest stable release is **v2.0.0**.

Download the release for your platform from the [Releases](https://github.com/xadomas2012/Plane-crazy-multitool/releases) page.

Available builds:

* Linux x64
* Windows x64
* macOS x64
* macOS arm64

## Linux

Download the Linux x64 ZIP and extract it.

Then run:

```bash
cd PC-Multitool
./install.sh
```

The installer places the application in:

```text
~/.local/opt/PC-Multitool/
```

and creates:

```text
~/.local/bin/pc-multitool
~/.local/share/applications/pc-multitool.desktop
```

That means the application can be launched from application menus such as **Rofi**.

You can also launch it directly:

```bash
pc-multitool
```

If `~/.local/bin` is not in your `PATH`, add:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Updating

Released versions include a built-in updater.

When a newer release is available, the application can download the appropriate platform package and use the bundled updater to replace the installed executable.

Stable releases can update normally, and prerelease builds can update through newer prereleases and into a newer stable release.

## Windows

Download the Windows x64 ZIP from the latest release and run:

```text
PC-Gear-Calculator.exe
```

Windows SmartScreen may display a warning because the executable is not code-signed.

## macOS

Download the package matching your Mac:

* **Intel:** macOS x64
* **Apple Silicon:** macOS arm64

Extract the ZIP and run the application.

## Themes

Included themes:

* Catppuccin Mocha
* Nord
* Gruvbox
* Solid

Accent colors can also be customized from the application.
