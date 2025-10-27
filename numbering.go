package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var dir string
var prefix string
var sortBy string
var reverse bool
var startNum int
var numberWidth int
var zeroPad bool
var targetExts string

func main() {
	// コマンドライン引数を明示
	dirPtr := flag.String("dir", "", "対象ディレクトリのパス")
	prefixPtr := flag.String("prefix", "", "共通の接頭辞")
	sortPtr := flag.String("sort", "name", "ソート順序 (name: 名前順, time: 更新日時順, exif: 撮影日時順)")
	reversePtr := flag.Bool("reverse", false, "逆順にソート")
	startPtr := flag.Int("start", 1, "採番の開始番号")
	widthPtr := flag.Int("width", 3, "採番の桁数")
	padPtr := flag.Bool("pad", true, "ゼロ埋めするかどうか")
	extsPtr := flag.String("exts", "jpeg", "対象拡張子 (jpeg,raw,heif など。カンマ区切り)")

	// 入力をパース
	flag.Parse()

	// 引数が正しくない場合は実行方法を明示
	if *dirPtr == "" || *prefixPtr == "" {
		log.Fatal("使用方法: go run numbering.go -dir=<ディレクトリ> -prefix=<接頭辞> -sort=<ソート順序> -reverse=<逆順にソート> -start=<開始番号> -width=<桁数> -pad=<ゼロ埋め> -exts=<対象拡張子>")
	}

	dir = *dirPtr
	prefix = *prefixPtr
	sortBy = *sortPtr
	reverse = *reversePtr
	startNum = *startPtr
	numberWidth = *widthPtr
	zeroPad = *padPtr
	targetExts = *extsPtr

	// 対象ディレクトリの中の全ファイルを取得([]fs.DirEntry)
	files, err := getFiles(dir)

	if err != nil {
		log.Fatal(err)
	}

	// 取得したファイルから画像/RAWファイルのみを抽出([]fs.DirEntry)
	imageFiles := extractImageFiles(files)

	// ファイルをソートしてからリネーム
	sortFiles(imageFiles)
	renameFiles(imageFiles)
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
func extractImageFiles(files []fs.DirEntry) []fs.DirEntry {
	var imageFiles []fs.DirEntry

	// 対象拡張子のマップを作成
	targetMap := make(map[string]bool)
	extGroups := strings.Split(targetExts, ",")
	for _, group := range extGroups {
		group = strings.TrimSpace(strings.ToLower(group))
		switch group {
		case "jpeg":
			targetMap[".jpeg"] = true
			targetMap[".jpg"] = true
		case "raw":
			targetMap[".cr2"] = true
			targetMap[".cr3"] = true
			targetMap[".nef"] = true
			targetMap[".arw"] = true
			targetMap[".raf"] = true
			targetMap[".rw2"] = true
			targetMap[".orf"] = true
			targetMap[".dng"] = true
		case "heif":
			targetMap[".heic"] = true
			targetMap[".heif"] = true
		}
	}

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if targetMap[ext] {
			imageFiles = append(imageFiles, file)
		}
	}

	return imageFiles
}

// used by main()
func renameFiles(jpegFiles []fs.DirEntry) {
	num := startNum
	successCount := 0
	for _, jpegFile := range jpegFiles {
		originalFileName := jpegFile.Name()
		oldPath := filepath.Join(dir, jpegFile.Name())
		ext := filepath.Ext(jpegFile.Name()) // 拡張子を取得

		// 採番の文字列を生成（桁数とゼロ埋めはオプション指定）
		var numStr string
		if zeroPad {
			numStr = fmt.Sprintf("%0*d", numberWidth, num)
		} else {
			numStr = fmt.Sprintf("%d", num)
		}
		newPath := filepath.Join(dir, fmt.Sprintf("%s%s%s", prefix, numStr, ext))
		newFileName := fmt.Sprintf("%s%s%s", prefix, numStr, ext)

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Println("Failed", oldPath, "->", newPath, err)
		} else {
			fmt.Printf("success: %s -> %s\n", originalFileName, newFileName)
			successCount++
		}
		num++
	}

	fmt.Printf("Completed: %d files renamed\n", successCount)
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
