package writter

import (
	"os"

	"github.com/ncw/directio"
)

// fileWriterAt 实现 io.WriteCloser，支持指定文件偏移的写入
func NewfileWriterAt(file *os.File, offset int64) *FileWriter {
	return &FileWriter{file: file, offset: offset}
}

type FileWriter struct {
	file   *os.File
	offset int64
}

func (w *FileWriter) Write(p []byte) (int, error) {
	// 写入指定位置
	alignedData := directio.AlignedBlock(len(p))
	copy(alignedData, p)

	n, err := w.file.WriteAt(alignedData[:len(p)], w.offset)
	w.offset += int64(n) // 更新偏移量
	return n, err
}

func (w *FileWriter) Close() error {
	return nil // 这里无需关闭文件，文件在外部管理
}
