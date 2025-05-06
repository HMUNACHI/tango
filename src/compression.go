package tango

import (
	"bytes"
	"io"
	"log"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/encoding"
)

// zstdCodec implements the gRPC encoding.Compressor interface using Zstandard compression.
type zstdCodec struct {
	level zstd.EncoderLevel
}

// Name returns the name of the codec.
func (z *zstdCodec) Name() string {
	return "zstd"
}

// Compress returns a WriteCloser that compresses data written to it.
func (z *zstdCodec) Compress(w io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(w, zstd.WithEncoderLevel(z.level))
}

// Decompress returns a Reader that decompresses data read from it.
func (z *zstdCodec) Decompress(r io.Reader) (io.Reader, error) {
	return zstd.NewReader(r)
}

// init registers the zstdCodec with gRPC's encoding package.
// The default compression level is set to SpeedDefault for a good balance
// between compression ratio and speed.
func init() {
	codec := &zstdCodec{level: zstd.SpeedDefault}
	encoding.RegisterCompressor(codec)
}

// PrintCompressionStats compresses the provided data and logs statistics about the compression.
// It returns the compression ratio achieved with the default compression level.
func PrintCompressionStats(data []byte) error {
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		w.Close()
		return err
	}
	w.Close()

	originalSize := len(data)
	compressedSize := buf.Len()
	compressionRate := float64(compressedSize) / float64(originalSize) * 100.0

	log.Printf("Original size: %d bytes, Compressed size: %d bytes, Compression rate: %.2f%%",
		originalSize, compressedSize, compressionRate)
	return nil
}
