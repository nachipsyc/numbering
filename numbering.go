package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var dir string
var prefix string
var sortBy string
var reverse bool

func main() {
	// コマンドライン引数を明示
	dirPtr := flag.String("dir", "", "対象ディレクトリのパス")
	prefixPtr := flag.String("prefix", "", "共通の接頭辞")
	sortPtr := flag.String("sort", "name", "ソート順序 (name: 名前順, time: 更新日時順, exif: 撮影日時順)")
	reversePtr := flag.Bool("reverse", false, "逆順にソート")

	// 入力をパース
	flag.Parse()

	// 引数が正しくない場合は実行方法を明示
	if *dirPtr == "" || *prefixPtr == "" {
		log.Fatal("使用方法: go run numbering.go -dir=<ディレクトリ> -prefix=<接頭辞>")
	}

	dir = *dirPtr
	prefix = *prefixPtr
	sortBy = *sortPtr
	reverse = *reversePtr

	// 対象ディレクトリの中の全ファイルを取得([]fs.DirEntry)
	files, err := getFiles(dir)

	if err != nil {
		log.Fatal(err)
	}

	// 取得したファイルからJPEGファイルのみを抽出([]fs.DirEntry)
	jpegFiles := extractJpegFiles(files)

	// ファイルをソートしてからリネーム
	sortFiles(jpegFiles)
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
		originalFileName := jpegFile.Name()
		oldPath := filepath.Join(dir, jpegFile.Name())
		ext := filepath.Ext(jpegFile.Name()) // 拡張子を取得

		// 3桁ゼロ埋め（例: "photo_001.jpg"）
		newPath := filepath.Join(dir, fmt.Sprintf("%s%03d%s", prefix, num, ext))
		newFileName := fmt.Sprintf("%s%03d%s", prefix, num, ext)

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Println("Failed", oldPath, "->", newPath, err)
		} else {
			fmt.Printf("success: %s -> %s\n", originalFileName, newFileName)
		}
		num++
	}

	log.Println("rename completed!!")
}

// getExifDateTime は写真のExif情報から撮影日時を取得します
func getExifDateTime(filepath string) (time.Time, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, err
	}

	return x.DateTime()
}

// 新しい関数: ファイルのソート
func sortFiles(files []fs.DirEntry) {
	// ソート関数を定義
	var less func(i, j int) bool

	switch sortBy {
	case "exif":
		less = func(i, j int) bool {
			timeI, errI := getExifDateTime(filepath.Join(dir, files[i].Name()))
			timeJ, errJ := getExifDateTime(filepath.Join(dir, files[j].Name()))

			if errI != nil || errJ != nil {
				infoI, _ := files[i].Info()
				infoJ, _ := files[j].Info()
				return infoI.ModTime().Before(infoJ.ModTime())
			}

			return timeI.Before(timeJ)
		}
	case "time":
		less = func(i, j int) bool {
			infoI, _ := files[i].Info()
			infoJ, _ := files[j].Info()
			return infoI.ModTime().Before(infoJ.ModTime())
		}
	default:
		less = func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		}
	}

	// reverseが指定されている場合は比較関数を反転
	if reverse {
		originalLess := less
		less = func(i, j int) bool {
			return !originalLess(i, j)
		}
	}

	sort.Slice(files, less)
}
