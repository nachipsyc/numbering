package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

var dir string
var prefix string

func main() {
	// コマンドライン引数を明示
	dirPtr := flag.String("dir", "", "対象ディレクトリのパス")
	prefixPtr := flag.String("prefix", "", "共通の接頭辞")

	// 入力をパース
	flag.Parse()

	// 引数が正しくない場合は実行方法を明示
	if *dirPtr == "" || *prefixPtr == "" {
		log.Fatal("使用方法: go run numbering.go -dir=<ディレクトリ> -prefix=<接頭辞>")
	}

	dir = *dirPtr
	prefix = *prefixPtr

	// 対象ディレクトリの中の全ファイルを取得([]fs.DirEntry)
	files, err := getFiles(dir)

	if err != nil {
		log.Fatal(err)
	}

	// 取得したファイルからJPEGファイルのみを抽出([]fs.DirEntry)
	jpegFiles := extractJpegFiles(files)

	// ファイルをリネーム
	renameFiles(jpegFiles)
}

// used by main()
func getFiles(sourceDir string) ([]fs.DirEntry, error) {
	// 対象ディレクトリの中のファイル全てを取得、格納
	files, err := os.ReadDir(sourceDir)

	// エラーがあれば"err"を返す
	if err != nil {
		return nil, err
	}

	return files, nil
}

// used by main()
func extractJpegFiles(files []fs.DirEntry) []fs.DirEntry {
	var jpegImages []fs.DirEntry
	for _, file := range files {
		switch filepath.Ext(file.Name()) {
		case ".jpeg", ".jpg", ".JPG":
			jpegImages = append(jpegImages, file)
		default:
		}
	}

	return jpegImages
}

// used by main()
func renameFiles(jpegFiles []fs.DirEntry) {
	num := 1
	for _, jpegFile := range jpegFiles {
		oldPath := filepath.Join(dir, jpegFile.Name())
		ext := filepath.Ext(jpegFile.Name()) // 拡張子を取得

		// 3桁ゼロ埋め（例: "photo_001.jpg"）
		newPath := filepath.Join(dir, fmt.Sprintf("%s%03d%s", prefix, num, ext))

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Println("Failed to rename:", oldPath, "->", newPath, err)
		}
		num++
	}

	log.Println("rename completed!!")
}
