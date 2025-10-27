package main

import (
	"encoding/json"
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

// RenameRecord はリネーム記録を表す構造体
type RenameRecord struct {
	OriginalName string `json:"original_name"`
	NewName      string `json:"new_name"`
}

// RenameLog はリネームログ全体を表す構造体
type RenameLog struct {
	Directory string         `json:"directory"`
	Records   []RenameRecord `json:"records"`
}

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
	undoPtr := flag.Bool("undo", false, "最後のリネーム処理をやり直す")

	// 入力をパース
	flag.Parse()

	// やり直し処理の場合
	if *undoPtr {
		undoRename()
		return
	}

	// 引数が正しくない場合は実行方法を明示
	if *dirPtr == "" || *prefixPtr == "" {
		log.Fatal("使用方法: go run numbering.go -dir=<ディレクトリ> -prefix=<接頭辞> -sort=<ソート順序> -reverse=<逆順にソート> -start=<開始番号> -width=<桁数> -pad=<ゼロ埋め> -exts=<対象拡張子>\nやり直し: go run numbering.go -undo")
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
	var renameLog RenameLog
	renameLog.Directory = dir
	renameLog.Records = make([]RenameRecord, 0)

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

			// リネーム記録を追加
			record := RenameRecord{
				OriginalName: originalFileName,
				NewName:      newFileName,
			}
			renameLog.Records = append(renameLog.Records, record)
		}
		num++
	}

	// リネームログをファイルに保存
	if successCount > 0 {
		saveRenameLog(renameLog)
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

// saveRenameLog はリネームログをJSONファイルに保存します
func saveRenameLog(renameLog RenameLog) {
	// 現在の作業ディレクトリを取得
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("作業ディレクトリの取得に失敗しました: %v", err)
		return
	}

	// logsディレクトリを作成
	logsDir := filepath.Join(workDir, "logs")
	err = os.MkdirAll(logsDir, 0755)
	if err != nil {
		log.Printf("logsディレクトリの作成に失敗しました: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logsDir, fmt.Sprintf("numbering_log_%s.json", timestamp))

	data, err := json.MarshalIndent(renameLog, "", "  ")
	if err != nil {
		log.Printf("ログの保存に失敗しました: %v", err)
		return
	}

	err = os.WriteFile(logFile, data, 0644)
	if err != nil {
		log.Printf("ログファイルの書き込みに失敗しました: %v", err)
		return
	}

	fmt.Printf("リネームログを保存しました: %s\n", logFile)
}

// findLatestLogFile は最新のログファイルを見つけます
func findLatestLogFile() (string, error) {
	// 現在の作業ディレクトリを取得
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("作業ディレクトリの取得に失敗しました: %v", err)
	}

	// logsディレクトリのパス
	logsDir := filepath.Join(workDir, "logs")

	files, err := filepath.Glob(filepath.Join(logsDir, "numbering_log_*.json"))
	if err != nil {
		return "", fmt.Errorf("ログファイルの検索に失敗しました: %v", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("ログファイルが見つかりません")
	}

	// ファイル名のタイムスタンプ部分でソートして最新を取得
	sort.Strings(files)
	return files[len(files)-1], nil
}

// loadRenameLog は最新のリネームログをJSONファイルから読み込みます
func loadRenameLog() (*RenameLog, error) {
	logFile, err := findLatestLogFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("ログファイルの読み込みに失敗しました: %v", err)
	}

	var renameLog RenameLog
	err = json.Unmarshal(data, &renameLog)
	if err != nil {
		return nil, fmt.Errorf("ログファイルの解析に失敗しました: %v", err)
	}

	return &renameLog, nil
}

// undoRename は最後のリネーム処理をやり直します
func undoRename() {
	renameLog, err := loadRenameLog()
	if err != nil {
		log.Fatal(err)
	}

	if len(renameLog.Records) == 0 {
		log.Fatal("やり直しできるリネーム記録がありません")
	}

	successCount := 0
	for _, record := range renameLog.Records {
		oldPath := filepath.Join(renameLog.Directory, record.NewName)
		newPath := filepath.Join(renameLog.Directory, record.OriginalName)

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Printf("Failed: %s -> %s (%v)", record.NewName, record.OriginalName, err)
		} else {
			fmt.Printf("success: %s -> %s\n", record.NewName, record.OriginalName)
			successCount++
		}
	}

	fmt.Printf("Completed: %d files restored\n", successCount)
}
