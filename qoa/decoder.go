package qoa

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"
)

// DecodeHeader decodes the QOA header and initializes the QOA struct with header information.
func DecodeHeader(bytes []byte) (*QOA, error) {
	size := len(bytes)
	if size < QOAMinFilesize {
		return nil, errors.New("qoa: file too small")
	}

	// Read the file header, verify the magic number ('qoaf') and read the total number of samples.
	fileHeader := binary.BigEndian.Uint64(bytes)

	if (fileHeader >> 32) != QOAMagic {
		return nil, errors.New("qoa: invalid magic number")
	}

	Samples := uint32(fileHeader & 0xffffffff)
	if Samples == 0 {
		return nil, errors.New("qoa: no samples found")
	}

	// Peek into the first frame header to get the number of channels and the SampleRate.
	frameHeader := binary.BigEndian.Uint64(bytes[8:])
	Channels := uint32(frameHeader>>56) & 0xff
	SampleRate := uint32(frameHeader>>32) & 0xffffff

	if Channels == 0 || SampleRate == 0 {
		return nil, errors.New("qoa: first frame header is invalid")
	}

	return &QOA{
		Samples:    Samples,
		Channels:   Channels,
		SampleRate: SampleRate,
	}, nil
}

// decodeFrame decodes a QOA frame and returns the size of the decoded frame.
func (q *QOA) decodeFrame(bytes []byte, size uint, sampleData []int16, frameLen *uint32) (uint, error) {
	if size < 8+QOALMSLen*4*uint(q.Channels) {
		return 0, errors.New("decodeFrame: too small")
	}

	p := uint(0)
	*frameLen = 0

	// Read and verify the frame header
	frameHeader := binary.BigEndian.Uint64(bytes[:8])
	p += 8
	channels := uint32((frameHeader >> 56) & 0x000000FF)
	sampleRate := uint32((frameHeader >> 32) & 0x00FFFFFF)
	samples := uint32((frameHeader >> 16) & 0x0000FFFF)
	frameSize := uint(frameHeader & 0x0000FFFF)

	dataSize := int(frameSize) - 8 - QOALMSLen*4*int(channels)
	numSlices := dataSize / 8
	maxTotalSamples := numSlices * QOASliceLen

	if channels != q.Channels ||
		sampleRate != q.SampleRate ||
		frameSize > size ||
		int(samples*channels) > maxTotalSamples {
		return 0, errors.New("decodeFrame: invalid header")
	}

	// Read the LMS state: 4 x 2 bytes history and 4 x 2 bytes weights per channel
	for c := uint32(0); c < channels; c++ {
		history := binary.BigEndian.Uint64(bytes[p:])
		weights := binary.BigEndian.Uint64(bytes[p+8:])
		p += 16

		for i := 0; i < QOALMSLen; i++ {
			q.lms[c].History[i] = int16(history >> 48)
			history <<= 16
			q.lms[c].Weights[i] = int16(weights >> 48)
			weights <<= 16
		}
	}

	// Decode all slices for all channels in this frame
	for sampleIndex := uint32(0); sampleIndex < samples; sampleIndex += QOASliceLen {
		for c := uint32(0); c < channels; c++ {
			slice := binary.BigEndian.Uint64(bytes[p:])
			p += 8

			scaleFactor := (slice >> 60) & 0xF
			slice <<= 4
			sliceStart := sampleIndex*channels + c
			sliceEnd := uint32(clamp(int(sampleIndex)+QOASliceLen, 0, int(samples)))*channels + c

			for si := sliceStart; si < sliceEnd; si += channels {
				predicted := q.lms[c].predict()
				quantized := int((slice >> 61) & 0x7)
				dequantized := qoaDequantTable[scaleFactor][quantized]
				reconstructed := clampS16(predicted + int(dequantized))

				sampleData[si] = reconstructed
				slice <<= 3

				q.lms[c].update(reconstructed, dequantized)
			}
		}
	}

	*frameLen = samples
	return p, nil
}

type Decoder struct {
	r        *bufio.Reader
	header   *QOA
	framebuf []byte
	samples  []int16
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	bufread := bufio.NewReaderSize(r, 32*1024)

	buf, err := bufread.Peek(16)
	if err != nil {
		return nil, fmt.Errorf("read qoa header: %w", err)
	}

	header, err := DecodeHeader(buf[:])
	if err != nil {
		return nil, fmt.Errorf("parse qua header: %w", err)
	}

	// discard the header bytes. no error handling needed, we peeked 16 bytes, so
	// we are guaranteed to be able to discard 8 bytes
	_, _ = bufread.Discard(8)

	// calculate size of one frame
	const sliceSize = 8
	const headerSize = 8
	const encoderState = 16
	frameSize := headerSize + (encoderState+sliceSize*QOASlicesPerFrame)*header.Channels

	dec := &Decoder{
		r:        bufread,
		header:   header,
		framebuf: make([]byte, frameSize),
		samples:  make([]int16, QOASlicesPerFrame*QOASliceLen*header.Channels),
	}

	return dec, err
}

func (d *Decoder) AppendSamples(samples []int16) ([]int16, error) {
	_, err := io.ReadFull(d.r, d.framebuf)
	if err != nil {
		return nil, fmt.Errorf("read frame: %w", err)
	}

	var sampleCount uint32

	_, err = d.header.decodeFrame(d.framebuf, uint(len(d.framebuf)), d.samples, &sampleCount)
	if err != nil {
		return nil, fmt.Errorf("decode frame: %w", err)
	}

	samples = append(samples, d.samples[:int(sampleCount*d.header.Channels)]...)
	return samples, nil
}

// Decode decodes the provided QOA encoded bytes and returns the QOA struct and the decoded audio sample data.
func Decode(bytes []byte) (*QOA, []int16, error) {
	q, err := DecodeHeader(bytes)
	if err != nil {
		return nil, nil, err
	}

	size := len(bytes)
	p := 8

	// Calculate the required size of the sample buffer and allocate
	totalSamples := q.Samples * q.Channels
	sampleData := make([]int16, totalSamples)

	sampleIndex := uint32(0)
	frameLen := uint32(0)
	frameSize := uint(0)

	// Decode all frames
	for {
		samplePtr := sampleData[sampleIndex*q.Channels:]
		frameSize, err = q.decodeFrame(bytes[p:], uint(size-p), samplePtr, &frameLen)
		if err != nil {
			return nil, nil, err
		}

		p += int(frameSize)
		sampleIndex += frameLen

		if !(frameSize > 0 && sampleIndex < q.Samples) {
			break
		}
	}

	q.Samples = sampleIndex
	return q, sampleData, nil
}

type qoaStream struct {
	decoder  *Decoder
	buffered []byte
	samples  []int16
}

func NewStream(r *Decoder) io.Reader {
	return &qoaStream{decoder: r}
}

func (q *qoaStream) Read(buf []byte) (n int, err error) {
	// only ever read an even number of bytes
	amount := len(buf) & ^1
	buf = buf[:amount]

	for {
		// take some samples from the buffer if available
		if n := min(len(buf), len(q.buffered)); n > 0 {
			copy(buf[:n], q.buffered[:n])
			q.buffered = q.buffered[n:]

			return n, nil
		}

		// buffer is empty, decode some samples
		q.samples, err = q.decoder.AppendSamples(q.samples[:0])
		if err != nil {
			return 0, fmt.Errorf("decode: %w", err)
		}

		// copy samples to buffer
		q.buffered = append(q.buffered, bytesViewOf(q.samples)...)
	}
}

func bytesViewOf(samples []int16) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&samples[0])), len(samples)*2)
}
