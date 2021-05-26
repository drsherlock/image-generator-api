package compress

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipFiles compresses one or many files into a single zip archive file.
// Param 1: filename is the output zip file's name.
// Param 2: files is a list of files to add to the zip.
func ZipFiles(fileName string, filesPath string) error {
	newZipFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipw := zip.NewWriter(newZipFile)
	defer zipw.Close()

	// Add files to zip
	err = filepath.Walk(filesPath, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			if err = AddFileToZip(filesPath+"/"+fileInfo.Name(), zipw); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func AddFileToZip(fileName string, zipw *zip.Writer) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("Failed to open %s: %s", fileName, err)
	}
	defer file.Close()

	wr, err := zipw.Create(filepath.Base(fileName))
	if err != nil {
		msg := "Failed to create entry for %s in zip file: %s"
		return fmt.Errorf(msg, fileName, err)
	}

	if _, err := io.Copy(wr, file); err != nil {
		return fmt.Errorf("Failed to write %s to zip: %s", fileName, err)
	}

	return nil
}
