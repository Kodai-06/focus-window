package main

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows APIで使用する定数
const (
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_POPUP          = 0x80000000
	LWA_ALPHA         = 0x00000002
	SM_CXSCREEN       = 0
	SM_CYSCREEN       = 1
	SW_SHOW           = 5
	RGN_DIFF          = 4
	BLACK_BRUSH       = 4
	WM_DESTROY        = 0x0002

	// ホットキー用の定数
	WM_HOTKEY   = 0x0312
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	VK_Q        = 0x51 // Qキー
	VK_C        = 0x43 // Cキー

	// ホットキーのID
	HOTKEY_QUIT   = 1
	HOTKEY_CENTER = 2

	// ウィンドウ操作・DWM用の定数
	DWMWA_EXTENDED_FRAME_BOUNDS = 9
	SWP_NOSIZE                  = 0x0001
	SWP_NOZORDER                = 0x0004
)

// DLLの読み込みとAPIの定義
var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll") // DWM API

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procCreateWindowExW            = user32.NewProc("CreateWindowExW")
	procRegisterClassExW           = user32.NewProc("RegisterClassExW")
	procDefWindowProcW             = user32.NewProc("DefWindowProcW")
	procPostQuitMessage            = user32.NewProc("PostQuitMessage")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procSetWindowRgn               = user32.NewProc("SetWindowRgn")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey           = user32.NewProc("UnregisterHotKey")
	procSetWindowPos               = user32.NewProc("SetWindowPos") // ウィンドウ移動用

	procCreateRectRgn  = gdi32.NewProc("CreateRectRgn")
	procCombineRgn     = gdi32.NewProc("CombineRgn")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procGetStockObject = gdi32.NewProc("GetStockObject")

	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute") // 影を除外した座標取得用
)

// 構造体の定義
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MSG struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

func main() {
	runtime.LockOSThread()

	fmt.Println("起動しました。")
	fmt.Println("[Ctrl + Shift + Q] : プログラムを終了")
	fmt.Println("[Ctrl + Shift + C] : アクティブウィンドウを画面中央に移動")

	w, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	screenWidth, screenHeight := int32(w), int32(h)

	hInst, _, _ := procGetModuleHandleW.Call(0)
	hInstance := windows.Handle(hInst)
	className, _ := windows.UTF16PtrFromString("ModernDarkOverlay")

	bgBrush, _, _ := procGetStockObject.Call(BLACK_BRUSH)
	wndClass := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   windows.NewCallback(wndProc),
		HInstance:     hInstance,
		LpszClassName: className,
		HbrBackground: windows.Handle(bgBrush),
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

	// グローバル変数の代わりにウィンドウハンドルを渡すための工夫は省略し、HWNDはそのまま使う
	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(WS_EX_TOPMOST|WS_EX_LAYERED|WS_EX_TRANSPARENT|WS_EX_TOOLWINDOW),
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(WS_POPUP),
		0, 0, uintptr(screenWidth), uintptr(screenHeight),
		0, 0, uintptr(hInstance), 0,
	)

	// ホットキーの登録
	procRegisterHotKey.Call(hwnd, HOTKEY_QUIT, MOD_CONTROL|MOD_SHIFT, VK_Q)
	procRegisterHotKey.Call(hwnd, HOTKEY_CENTER, MOD_CONTROL|MOD_SHIFT, VK_C)
	defer procUnregisterHotKey.Call(hwnd, HOTKEY_QUIT)
	defer procUnregisterHotKey.Call(hwnd, HOTKEY_CENTER)

	// 不透明度はお好みで変更(現在は真っ暗の255)
	procSetLayeredWindowAttributes.Call(hwnd, 0, 255, LWA_ALPHA)
	procShowWindow.Call(hwnd, SW_SHOW)

	go func() {
		var prevHwnd uintptr
		var prevRect RECT

		for {
			time.Sleep(100 * time.Millisecond)

			fgHwnd, _, _ := procGetForegroundWindow.Call()
			if fgHwnd == 0 || fgHwnd == hwnd {
				continue
			}

			var rect RECT
			// ★変更: DwmGetWindowAttributeを使って「影を除外した実際の見た目の座標」を取得
			ret, _, _ := procDwmGetWindowAttribute.Call(
				fgHwnd,
				uintptr(DWMWA_EXTENDED_FRAME_BOUNDS),
				uintptr(unsafe.Pointer(&rect)),
				uintptr(unsafe.Sizeof(rect)),
			)

			// DWMが効かない古いウィンドウなどの場合は従来のGetWindowRectにフォールバック
			if ret != 0 {
				procGetWindowRect.Call(fgHwnd, uintptr(unsafe.Pointer(&rect)))
			}

			if fgHwnd != prevHwnd || rect != prevRect {
				screenRgn, _, _ := procCreateRectRgn.Call(0, 0, uintptr(screenWidth), uintptr(screenHeight))
				holeRgn, _, _ := procCreateRectRgn.Call(uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right), uintptr(rect.Bottom))

				resultRgn, _, _ := procCreateRectRgn.Call(0, 0, 0, 0)
				procCombineRgn.Call(resultRgn, screenRgn, holeRgn, RGN_DIFF)

				procSetWindowRgn.Call(hwnd, resultRgn, 1)

				procDeleteObject.Call(screenRgn)
				procDeleteObject.Call(holeRgn)

				prevHwnd = fgHwnd
				prevRect = rect
			}
		}
	}()

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func wndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_HOTKEY:
		id := wParam
		if id == HOTKEY_QUIT {
			// 終了
			procPostQuitMessage.Call(0)
		} else if id == HOTKEY_CENTER {
			// ★追加: アクティブウィンドウを中央に移動
			fgHwnd, _, _ := procGetForegroundWindow.Call()
			if fgHwnd != 0 && fgHwnd != uintptr(hwnd) {
				var rect RECT
				ret, _, _ := procDwmGetWindowAttribute.Call(fgHwnd, uintptr(DWMWA_EXTENDED_FRAME_BOUNDS), uintptr(unsafe.Pointer(&rect)), uintptr(unsafe.Sizeof(rect)))
				if ret != 0 {
					procGetWindowRect.Call(fgHwnd, uintptr(unsafe.Pointer(&rect)))
				}

				// ウィンドウの幅と高さを計算
				width := rect.Right - rect.Left
				height := rect.Bottom - rect.Top

				// 画面サイズを再取得
				sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
				sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

				// 中央になる座標を計算
				newX := (int32(sw) - width) / 2
				newY := (int32(sh) - height) / 2

				// ウィンドウを移動 (サイズとZオーダーは変更しない)
				procSetWindowPos.Call(fgHwnd, 0, uintptr(newX), uintptr(newY), 0, 0, uintptr(SWP_NOSIZE|SWP_NOZORDER))
			}
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}
