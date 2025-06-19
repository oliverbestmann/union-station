package main

import "io"

type StereoStream struct {
	source io.ReadSeeker
	buf    []byte
}

func NewStereoStream(source io.ReadSeeker) *StereoStream {
	return &StereoStream{
		source: source,
	}
}

func (s *StereoStream) Read(b []byte) (int, error) {
	l := len(b) / 8 * 8
	l /= 2

	if cap(s.buf) < l {
		s.buf = make([]byte, l)
	}

	n, err := s.source.Read(s.buf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}

	for i := 0; i < n/4; i++ {
		b[8*i] = s.buf[4*i]
		b[8*i+1] = s.buf[4*i+1]
		b[8*i+2] = s.buf[4*i+2]
		b[8*i+3] = s.buf[4*i+3]
		b[8*i+4] = s.buf[4*i]
		b[8*i+5] = s.buf[4*i+1]
		b[8*i+6] = s.buf[4*i+2]
		b[8*i+7] = s.buf[4*i+3]
	}

	n *= 2

	return n, err
}

func (s *StereoStream) Seek(offset int64, whence int) (int64, error) {
	offset = offset / 8 * 8
	offset /= 2

	return s.source.Seek(offset, whence)
}
