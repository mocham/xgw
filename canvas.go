package main
func simpleCanvasWidget(title string, img RGBAData) {
	top, left := 0, 0
	universalWidget(title, 0, 0, width, height, func(ximg *XImage) (int, int) {
		if img.Width - left < width { left = img.Width - width }
		if left < 0 { left = 0 }
		if img.Height - top < height { top = img.Height - height }
		if top < 0 { top = 0 } 
		w, h := width, height
		if img.Width - left < w { w = img.Width - left }
		if img.Height - top < h { h = img.Height - top }
		ximg.XDraw(Crop(img, left, top, w, h), 0, 0)
		return 0, 0
	}, func (detail byte, x, y int16) int {
		switch detail {
		case 4: top -= 200
		case 5: top += 200
		case 6: left -= 200
		case 7: left += 200
		default: return 0
		}
		return 1
	}, func (detail byte) int {
		switch detail {
		case 24: return -1 //"q"
		case 111: top -= 200
		case 116: top += 200
		case 113: left -=200
		case 114: left +=200
		default: return 0
		}
		return 1
	}, nil, windowRaiseFocuser)
}

func multipleCanvasWidget(width, height int, title, imagePath string) {
	var images []string
	for _, file := range getFileList(imagePath) { if extInGroup("img", file) { images = append(images, file) } }
	if len(images) == 0 { return }
	cache, img_id, old_id, xOffset, numInput, cacheCount := map[int]RGBAData{}, 0, 0, 0, 0, 0
	draw := func (ximg *XImage, i int) (int, int) {
		if i < 0 || i >= len(images) { return 0, 0 }
		img, exists := cache[i]
		if !exists {
			if cacheCount += 1; cacheCount % 200 == 0 { cache = make(map[int]RGBAData) }
			var err error
			if img, err = getImageFromPath(images[i], width, height); err != nil { return 0, 0 } else { cache[i] = img }
		}
		if xOffset + img.Width > width { return 0, 0 }
		ximg.XDraw(img, xOffset, 0)
		if img.Height < height { ximg.XDraw(blankImage(img.Width, height - img.Height), xOffset, img.Height) }
		return img.Width, img.Height
	}
	universalWidget(title, 0, 0, width, height, func(ximg *XImage) (xoff, max_ht int) {
		xOffset, max_ht, old_id = 0, 100, img_id
		for {
			if img_id < 0 { img_id += len(images); continue }
			if img_id >= len(images) { img_id -= len(images); continue }
			delta, ht := draw(ximg, img_id)
			if delta == 0 { break }
			img_id += 1
			xOffset += delta
			if ht > max_ht { max_ht = ht }
		}
		xoff = xOffset
		return
	}, nil, func (detail byte) int {
		switch detail {
		case 10, 11, 12, 13, 14, 15, 16, 17, 18, 19: numInput = numInput * 10 + int(detail + 1) % 10; return 0
		case 36: if numInput > 0 { img_id, numInput  = numInput, 0 } // Enter
		case 24: return -1 //"q"
		case 113: if len(images) == 0 { return 0 }; img_id = old_id - 1
		case 114: if len(images) == 0 { return 0 }
		default: return 0
		}
		return 1
	}, nil, windowRaiseFocuser)
}

func scrotWidget() {
	_, data32 := screenshot(0, 0, width, height)
	state, scrImg := 0, RGBAData{Pix: data32, Stride: width*4, Width: width, Height: height}
	var coord [4]int
	universalWidget("auto-scrot", 0, 0, width, height, func(ximg *XImage) (int, int) {
		switch state {
		case 0: ximg.XDraw(scrImg, 0, 0)
		case 1: ximg.XDraw(blankImage(width, coord[1]), 0, 0); ximg.XDraw(blankImage(coord[0], height), 0, 0)
		}
		return 0, 0
	}, func (detail byte, x, y int16) int {
		coord[state*2], coord[state*2+1] = int(x), int(y)
		state += 1
		if state < 2 { return 1 }
		w, h := coord[2] - coord[0], coord[3] - coord[1]
		output := make([]uint32, 0, w*h)[:w*h]
		for i:=coord[1]; i < coord[3]; i++ { copy(output[(i-coord[1])*w:(i-coord[1]+1)*w], data32[i*width+coord[0]:(i+1)*width+coord[0]]) }
		savePNG(output, w, h, "/tmp/screenshot.png")
		return -1
	}, func (detail byte) int { return -1 }, nil, windowRaiseFocuser)
}

func recordWidget() {
	_, data32 := screenshot(0, 0, width, height)
	var coord [4]int
	state, stop, scrImg := 0, false, RGBAData{Pix: data32, Stride: width*4, Width: width, Height: height}
	universalWidget("auto-scrot", 0, 0, width, height, func(ximg *XImage) (int, int) {
		switch state {
		case 0: ximg.XDraw(scrImg, 0, 0)
		case 1: ximg.XDraw(blankImage(width, coord[1]), 0, 0); ximg.XDraw(blankImage(coord[0], height), 0, 0)
		case 2: ximg.XDraw(blankImage(glyphWidth*3, glyphHeight), 0, 0);; return glyphWidth*3, glyphHeight
		}
		return 0, 0
	}, func (detail byte, x, y int16) int {
		switch state {
		case 0, 1: coord[state*2], coord[state*2+1] = int(x), int(y); 
		case 3: go recordScreen("/tmp/record.webm", timeHour, coord[0], coord[1], coord[2]-coord[0], coord[3]-coord[1], 10, &stop)
		case 4: stop = true; return -1
		}
		state += 1; return 1
	}, nil, nil, windowRaiseFocuser)
}
