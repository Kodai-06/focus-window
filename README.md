# focus-window

`focus-window` is a Windows tool designed to help you concentrate as deeply as possible by darkening everything except the active window.
It also lets you instantly move the active window to the center of the screen using a shortcut key.

## How It Works

- Only the selected (active) window stays clearly visible.
- Background areas and inactive windows are covered in black.
- It runs completely in the background without opening a terminal or visible app window.

## Keyboard Shortcuts

While the app is running, the following global hotkeys are always available:

- `Ctrl + Shift + C`: Moves the currently active window to the exact center of the screen.
- `Ctrl + Shift + Q`: Exits `focus-window` (the screen returns to normal).

## Installation and Usage

### For General Users (Easiest)

1. Go to the **Releases** page and download the latest `focus-window.exe`.
2. Double-click the downloaded `.exe` file to launch it.
	(No visible app window appears; the feature becomes active immediately.)

### For Developers (Build from Source)

You need a Go environment (Go 1.16 or later recommended).

```bash
# Clone the repository
git clone https://github.com/Kodai-06/focus-window.git
cd focus-window

# Download dependencies
go mod tidy

# Build an .exe without showing a console window
go build -ldflags "-H windowsgui" -o focus-window.exe
```

