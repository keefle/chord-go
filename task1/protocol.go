package main

import (
	"encoding/binary"
	"io"
)

func sendMessage(conn io.Writer, msg []byte) error {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(len(msg)))

	if _, err := conn.Write(bs); err != nil {
		return err
	}
	if _, err := conn.Write(msg); err != nil {
		return err
	}

	return nil
}

func readMessage(conn io.Reader, wr io.Writer) error {
	bs := make([]byte, 8)
	conn.Read(bs)
	size := binary.BigEndian.Uint64(bs)
	buff := make([]byte, 4092)

	var recv uint64 = 0
	for {
		if (size - recv) < 4092 {
			buff := make([]byte, size-recv)

			if _, err := conn.Read(buff); err != nil {
				return err
			}

			if _, err := wr.Write(buff); err != nil {
				return err
			}

			return nil
		}

		n, err := conn.Read(buff)
		if err != nil {
			return err
		}

		if _, err := wr.Write(buff); err != nil {
			return err
		}

		recv += uint64(n)
	}

	return nil
}
