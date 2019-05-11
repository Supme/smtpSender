package smtpSender

import "io"

type DelimitWriter struct {
	n      int
	cnt    int
	dr     []byte
	writer io.Writer
}

func NewDelimitWriter(writer io.Writer, delimiter []byte, cnt int) *DelimitWriter {
	return &DelimitWriter{n: 0, cnt: cnt, dr: delimiter, writer: writer}
}

func (w *DelimitWriter) Write(p []byte) (int, error) {
	var err error
	for i := range p {
		_, err = w.writer.Write(p[i : i+1])
		if err != nil {
			break
		}
		if w.n++; w.n%w.cnt == 0 {
			_, err = w.writer.Write(w.dr)
			if err != nil {
				break
			}
		}
	}
	return w.n, err
}
