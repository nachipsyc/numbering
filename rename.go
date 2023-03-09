package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var dir string = ""
var main_title string = ""

func main() {
	// 入力をパース
	flag.Parse()

	// 対象ディレクトリのパスを引数から取得(string)
	dir = flag.Arg(0)

	// タイトルが指定された場合はセット
	main_title = flag.Arg(1)

	// 対象ディレクトリの中の全ファイルを取得([]os.FileInfo)
	files, err := getFiles(dir)

	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	// 取得したファイルからJPEGファイルのみを抽出([]os.FileInfo)
	jpeg_files := extractJpegFiles(files)

	// ファイルをリネーム
	renameFiles(jpeg_files, main_title)
}

// used by main()
func getFiles(source_dir string) ([]os.FileInfo, error) {
	// 対象ディレクトリの中のファイル全てを取得、格納
	files, err := ioutil.ReadDir(source_dir)

	// エラーがあれば"err"を返す
	if err != nil {
		return nil, err
	}

	return files, nil
}

// used by main()
func extractJpegFiles(files []os.FileInfo) []os.FileInfo {
	var jpeg_images []os.FileInfo
	for _, file := range files {
		switch filepath.Ext(file.Name()) {
		case ".jpeg", ".jpg", ".JPG":
			jpeg_images = append(jpeg_images, file)
		default:
		}
	}

	return jpeg_images
}

// used by main()
func renameFiles(jpeg_files []os.FileInfo, main_title string) {
	num := 1
	for _, jpeg_file := range jpeg_files {
		// ファイルを画像として読み込み
		decoded_image, _ := decodeImage(jpeg_file)

		if decoded_image != nil {
			// リサイズした画像を書き込み
			encodeImage(decoded_image, main_title, num)
		}
		num++
	}

	fmt.Println("rename completed!!")
}

// used by renameFiles()
func decodeImage(jpeg_file os.FileInfo) (image.Image, error) {
	io_file, err := conversionToReader(jpeg_file)
	if err != nil {
		return nil, err
	}

	decoded_image, _, err := image.Decode(io_file)
	if err != nil {
		return nil, err
	}
	return decoded_image, nil
}

// used by renameFiles()
func encodeImage(decoded_image image.Image, main_title string, number int) error {
	output, err := os.Create(dir + "/" + main_title + strconv.Itoa(number) + ".jpeg")
	if err != nil {
		return err
	}
	
	defer output.Close()
	
	opts := &jpeg.Options{Quality: 100}
	if err := jpeg.Encode(output, decoded_image, opts); err != nil {
		return err
	}
	
	return nil
}

// used by decodeImage()
func conversionToReader(jpeg_file os.FileInfo) (io.Reader, error) {
	io_file, err := os.Open(dir + "/" + jpeg_file.Name())
	if err != nil {
		return nil, err
	}
	return io_file, nil
}

