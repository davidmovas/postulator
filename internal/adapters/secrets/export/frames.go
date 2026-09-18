package export

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	stderrors "errors"
	"io"

	"github.com/davidmovas/postulator/internal/adapters/secrets/masterpassword"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Header    = "PSTX1"
	FrameSize = 64 * 1024

	prefixLength  = 4
	counterLength = 8
	lengthLength  = 4
	saltLength    = masterpassword.SaltLength
	preambleSize  = len(Header) + saltLength + prefixLength
	maxFrameBytes = FrameSize + 64

	frameContinues = 0
	frameFinal     = 1
)

type sealer struct {
	out     io.Writer
	aead    cipher.AEAD
	prefix  []byte
	counter uint64
	pending []byte
}

func newSealer(out io.Writer, password string) (*sealer, error) {
	if password == "" {
		return nil, errors.New(errors.Invalid, "the backup password must not be empty")
	}

	preamble := make([]byte, 0, preambleSize)
	preamble = append(preamble, Header...)

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the backup salt")
	}
	prefix := make([]byte, prefixLength)
	if _, err := rand.Read(prefix); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the backup nonce prefix")
	}

	aead, err := frameCipher(password, salt)
	if err != nil {
		return nil, err
	}

	preamble = append(preamble, salt...)
	preamble = append(preamble, prefix...)
	if _, err = out.Write(preamble); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "write the backup header")
	}
	return &sealer{out: out, aead: aead, prefix: prefix, pending: make([]byte, 0, FrameSize)}, nil
}

func (s *sealer) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		room := FrameSize - len(s.pending)
		take := min(room, len(p))
		s.pending = append(s.pending, p[:take]...)
		p = p[take:]
		written += take

		if len(s.pending) == FrameSize {
			if err := s.flush(frameContinues); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (s *sealer) Close() error {
	return s.flush(frameFinal)
}

func (s *sealer) flush(final byte) error {
	frame := make([]byte, 1+lengthLength, 1+lengthLength+len(s.pending)+s.aead.Overhead())
	frame[0] = final
	binary.BigEndian.PutUint32(frame[1:], uint32(len(s.pending)+s.aead.Overhead()))
	frame = s.aead.Seal(frame, nonce(s.prefix, s.counter), s.pending, associated(s.counter, final))

	if _, err := s.out.Write(frame); err != nil {
		return errors.Wrap(err, errors.Internal, "write a backup frame")
	}
	s.counter++
	s.pending = s.pending[:0]
	return nil
}

type opener struct {
	in      io.Reader
	aead    cipher.AEAD
	prefix  []byte
	counter uint64
	pending []byte
	done    bool
}

func newOpener(in io.Reader, password string) (*opener, error) {
	if password == "" {
		return nil, errors.New(errors.Invalid, "the backup password must not be empty")
	}

	preamble := make([]byte, preambleSize)
	if _, err := io.ReadFull(in, preamble); err != nil {
		return nil, errors.New(errors.Invalid, "the backup file is not readable").WithInternal(err)
	}
	if string(preamble[:len(Header)]) != Header {
		return nil, errors.New(errors.Invalid, "the backup file carries no "+Header+" header")
	}

	aead, err := frameCipher(password, preamble[len(Header):len(Header)+saltLength])
	if err != nil {
		return nil, err
	}
	return &opener{in: in, aead: aead, prefix: preamble[len(Header)+saltLength:]}, nil
}

func (o *opener) Read(p []byte) (int, error) {
	for len(p) > 0 && len(o.pending) == 0 {
		if o.done {
			return 0, io.EOF
		}
		if err := o.fill(); err != nil {
			return 0, err
		}
	}

	read := copy(p, o.pending)
	o.pending = o.pending[read:]
	return read, nil
}

func (o *opener) fill() error {
	head := make([]byte, 1+lengthLength)
	if _, err := io.ReadFull(o.in, head); err != nil {
		return errors.New(errors.Invalid, "the backup file ends inside a frame").WithInternal(err)
	}

	final := head[0]
	if final != frameContinues && final != frameFinal {
		return errors.New(errors.Invalid, "the backup file carries an unreadable frame")
	}

	size := binary.BigEndian.Uint32(head[1:])
	if size < uint32(o.aead.Overhead()) || size > maxFrameBytes {
		return errors.New(errors.Invalid, "the backup file carries a frame of an impossible size")
	}

	sealed := make([]byte, size)
	if _, err := io.ReadFull(o.in, sealed); err != nil {
		return errors.New(errors.Invalid, "the backup file ends inside a frame").WithInternal(err)
	}

	plain, err := o.aead.Open(sealed[:0], nonce(o.prefix, o.counter), sealed, associated(o.counter, final))
	if err != nil {
		return errors.New(errors.Invalid, "the backup cannot be read with that password").WithInternal(err)
	}

	o.counter++
	o.pending = plain
	o.done = final == frameFinal
	return nil
}

func (o *opener) complete() error {
	if !o.done {
		return errors.New(errors.Invalid, "the backup file is truncated")
	}
	if _, err := o.in.Read(make([]byte, 1)); !stderrors.Is(err, io.EOF) {
		return errors.New(errors.Invalid, "the backup file carries trailing bytes")
	}
	return nil
}

func frameCipher(password string, salt []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(masterpassword.Derive(password, salt))
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the backup cipher")
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the backup aead")
	}
	return aead, nil
}

func nonce(prefix []byte, counter uint64) []byte {
	out := make([]byte, 0, prefixLength+counterLength)
	out = append(out, prefix...)
	return binary.BigEndian.AppendUint64(out, counter)
}

func associated(counter uint64, final byte) []byte {
	out := make([]byte, 0, len(Header)+counterLength+1)
	out = append(out, Header...)
	out = binary.BigEndian.AppendUint64(out, counter)
	return append(out, final)
}
