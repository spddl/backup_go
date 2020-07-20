package archivex

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/monochromegane/go-gitignore"
)

// AddAllGitIgnore
func (z *ZipFile) AddAllGitIgnore(dir string, includeCurrentFolder bool, gitignore string) error {
	dir = path.Clean(dir)
	return addAllGitIgnore(dir, dir, includeCurrentFolder, gitignore, func(info os.FileInfo, file io.Reader, entryName string) (err error) {

		// Create a header based off of the fileinfo
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// If it's a file, set the compression method to deflate (leave directories uncompressed)
		if !info.IsDir() {
			header.Method = zip.Deflate
		}

		// Set the header's name to what we want--it may not include the top folder
		header.Name = entryName

		// Add a trailing slash if the entry is a directory
		if info.IsDir() {
			header.Name += "/"
		}

		// Get a writer in the archive based on our header
		writer, err := z.Writer.CreateHeader(header)
		if err != nil {
			return err
		}

		// If we have a file to write (i.e., not a directory) then pipe the file into the archive writer
		if file != nil {
			if _, err := io.Copy(writer, file); err != nil {
				return err
			}
		}

		return nil
	})
}

// addAllGitIgnore
func addAllGitIgnore(dir string, rootDir string, includeCurrentFolder bool, gitignorePath string, writerFunc ArchiveWriteFunc) error {
	gitignore, err := gitignore.NewGitIgnore(gitignorePath)
	if err != nil {
		return err
	}

	// Get a list of all entries in the directory, as []os.FileInfo
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	// Loop through all entries
	for _, info := range fileInfos {
		full := filepath.Join(dir, info.Name())

		if gitignore.Match(full, info.IsDir()) {
			continue
		}

		// If the entry is a file, get an io.Reader for it
		var file *os.File
		var reader io.Reader
		if !info.IsDir() {
			file, err = os.Open(full)
			if err != nil {
				return err
			}
			reader = file
		}

		// Write the entry into the archive
		subDir := getSubDir(dir, rootDir, includeCurrentFolder)
		entryName := path.Join(subDir, info.Name())
		if err := writerFunc(info, reader, entryName); err != nil {
			if file != nil {
				file.Close()
			}
			return err
		}

		if file != nil {
			if err := file.Close(); err != nil {
				return err
			}
		}

		// If the entry is a directory, recurse into it
		if info.IsDir() {
			addAllGitIgnore(full, rootDir, includeCurrentFolder, gitignorePath, writerFunc)
		}
	}

	return nil
}
