package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/user"
	"path/filepath"
	"unsafe"

	"github.com/lxn/win"
	"github.com/pkg/errors"
)

func takeScreenshot() (image.Image, error) {
	hwnd := win.GetDesktopWindow()
	var screenSize win.RECT
	win.GetClientRect(hwnd, &screenSize)

	fmt.Printf("%+v\n", screenSize)

	hdc := win.GetDC(hwnd)
	if hdc == 0 {
		return nil, errors.Errorf("GetDC failed")
	}
	defer win.ReleaseDC(hwnd, hdc)

	memDev := win.CreateCompatibleDC(hdc)
	if memDev == 0 {
		return nil, errors.Errorf("CreateCompatibleDC failed")
	}
	defer win.DeleteDC(memDev)

	bmp := win.CreateCompatibleBitmap(hdc, screenSize.Right, screenSize.Bottom)
	if bmp == 0 {
		return nil, errors.Errorf("CreateCompatibleBitmap failed")
	}
	defer win.DeleteObject(win.HGDIOBJ(bmp))

	header := win.BITMAPINFOHEADER{
		BiPlanes:      1,
		BiBitCount:    32,
		BiWidth:       int32(screenSize.Right),
		BiHeight:      int32(-screenSize.Bottom),
		BiCompression: win.BI_RGB,
		BiSizeImage:   0,
	}
	header.BiSize = uint32(unsafe.Sizeof(header))

	lineSize := (int64(screenSize.Right)*int64(header.BiBitCount) + 31) / 32
	bitmapDataSize := uintptr(lineSize * 4 * int64(screenSize.Bottom))
	dstMem := win.GlobalAlloc(win.GMEM_MOVEABLE, bitmapDataSize)
	defer win.GlobalFree(dstMem)
	dstPtr := win.GlobalLock(dstMem)
	defer win.GlobalUnlock(dstMem)

	old := win.SelectObject(memDev, win.HGDIOBJ(bmp))
	if old == 0 {
		return nil, errors.Errorf("SelectObject failed")
	}
	defer win.SelectObject(memDev, old)

	if !win.BitBlt(
		memDev, 0, 0, screenSize.Right, screenSize.Bottom,
		hdc, 0, 0, win.SRCCOPY) {
		return nil, errors.Errorf("BitBlt failed")
	}

	if win.GetDIBits(
		hdc, bmp, 0, uint32(screenSize.Bottom), (*uint8)(dstPtr),
		(*win.BITMAPINFO)(unsafe.Pointer(&header)), win.DIB_RGB_COLORS) == 0 {
		return nil, errors.Errorf("GetDIBits failed")
	}

	img := image.NewRGBA(image.Rect(
		0, 0, int(screenSize.Right), int(screenSize.Bottom)))
	i := 0
	ptr := uintptr(dstPtr)
	for y := int32(0); y < screenSize.Bottom; y++ {
		for x := int32(0); x < screenSize.Right; x++ {
			v0 := *(*uint8)(unsafe.Pointer(ptr))
			v1 := *(*uint8)(unsafe.Pointer(ptr + 1))
			v2 := *(*uint8)(unsafe.Pointer(ptr + 2))
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v2, v1, v0, 255
			i += 4
			ptr += 4
		}
	}

	return img, nil
}

func main() {
	outImg, err := takeScreenshot()
	if err != nil {
		fmt.Printf("screenshot taking error: %+v\n", err)
	}

	u, err := user.Current()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	outPath := filepath.Join(u.HomeDir, "ss.png")
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Println("Output file open error:", err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, outImg); err != nil {
		fmt.Println("Failed writing to png:", err)
		return
	}
}
