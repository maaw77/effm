package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Определяем флаги командной строки
	outputFile := flag.String("output", "project_report.txt", "Имя выходного файла")
	rootDir := flag.String("dir", ".", "Корневая директория для сканирования")
	extensions := flag.String("ext", "go,yaml,yml,sql,mod,sh", "Расширения файлов через запятую")
	excludeDirs := flag.String("exclude", "vendor,node_modules,.git,cmd/scanner", "Директории для исключения через запятую")

	flag.Parse()

	// Парсим расширения
	extList := strings.Split(*extensions, ",")
	for i, ext := range extList {
		extList[i] = strings.TrimSpace(ext)
		if !strings.HasPrefix(ext, ".") {
			extList[i] = "." + ext
		}
	}

	// Парсим исключаемые директории
	excludeList := strings.Split(*excludeDirs, ",")
	for i, dir := range excludeList {
		excludeList[i] = strings.TrimSpace(dir)
	}

	// Добавляем папку сканера в исключения, если её там нет
	scannerDir := "cmd/scanner"
	hasScannerDir := false
	for _, dir := range excludeList {
		if dir == scannerDir {
			hasScannerDir = true
			break
		}
	}
	if !hasScannerDir {
		excludeList = append(excludeList, scannerDir)
	}

	fmt.Printf("Сканирование проекта в директории: %s\n", *rootDir)
	fmt.Printf("Ищем файлы с расширениями: %s\n", strings.Join(extList, ", "))
	fmt.Printf("Исключаем директории: %s\n", strings.Join(excludeList, ", "))

	// Создаем выходной файл
	file, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("Ошибка создания файла: %v\n", err)
		return
	}
	defer file.Close()

	// Записываем заголовок
	header := fmt.Sprintf("Отчет по проекту: %s\n", *rootDir)
	header += fmt.Sprintf("Расширения: %s\n", strings.Join(extList, ", "))
	header += fmt.Sprintf("Исключенные директории: %s\n", strings.Join(excludeList, ", "))
	header += strings.Repeat("=", 80) + "\n\n"

	if _, err := file.WriteString(header); err != nil {
		fmt.Printf("Ошибка записи в файл: %v\n", err)
		return
	}

	// Сканируем директорию
	err = scanDirectory(*rootDir, *rootDir, extList, excludeList, file)
	if err != nil {
		fmt.Printf("Ошибка сканирования: %v\n", err)
		return
	}

	fmt.Printf("Отчет успешно сохранен в файл: %s\n", *outputFile)
}

// scanDirectory рекурсивно сканирует директорию
func scanDirectory(rootDir, currentDir string, extensions, excludeDirs []string, output io.Writer) error {
	return filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем корневую директорию
		if path == rootDir {
			return nil
		}

		// Проверяем, является ли путь исключенной директорией
		if info.IsDir() {
			dirName := filepath.Base(path)
			for _, exclude := range excludeDirs {
				if dirName == exclude {
					return filepath.SkipDir
				}
			}
			// Дополнительная проверка полного относительного пути
			relPath, err := filepath.Rel(rootDir, path)
			if err == nil {
				for _, exclude := range excludeDirs {
					if relPath == exclude {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		// Проверяем расширение файла
		ext := filepath.Ext(path)
		shouldInclude := false
		for _, allowedExt := range extensions {
			if ext == allowedExt {
				shouldInclude = true
				break
			}
		}

		if !shouldInclude {
			return nil
		}

		// Получаем относительный путь
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			relPath = path
		}

		// Исключаем сам файл сканера
		if relPath == "cmd/scanner/scanner.go" {
			return nil
		}

		// Записываем информацию о файле
		return processFile(relPath, path, output)
	})
}

// processFile обрабатывает один файл и записывает его в отчет
func processFile(relPath, fullPath string, output io.Writer) error {
	// Читаем содержимое файла
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла %s: %v", fullPath, err)
	}

	// Формируем блок для файла
	fileBlock := fmt.Sprintf("%s\n", relPath)
	fileBlock += fmt.Sprintf("// %s\n", filepath.Base(fullPath))
	fileBlock += fmt.Sprintf("// Расширение: %s\n\n", filepath.Ext(fullPath))
	fileBlock += string(content)
	fileBlock += "\n\n" + strings.Repeat("-", 80) + "\n\n"

	// Записываем в выходной поток
	if _, err := output.Write([]byte(fileBlock)); err != nil {
		return fmt.Errorf("ошибка записи: %v", err)
	}

	return nil
}
