/*
  	MordorRpBot — https://www.blast.hk/threads/72108/
    Copyright (C) 2020 RINWARES

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"time"
)

/* Contained in every file.
==============================FileHeader==============================
	Marker				= 16 byte
	Version				= 4 byte
	BuildTime			= 8 byte
==============================FileHeader==============================
*/

const fileHeaderSize = int64(16 + 4 + 8)

// Major, minor, patch.
const CurrentDatabaseVersion = uint32(1)<<24 | uint32(0)<<16 | uint32(0)<<8

type FileHeader struct {
	Marker    [16]byte
	Version   uint32
	BuildTime time.Time
}

func (m *FileHeader) MarshalBinary() ([]byte, error) {
	buff := make([]byte, fileHeaderSize)
	copy(buff[:16], m.Marker[:])
	binary.LittleEndian.PutUint32(buff[16:20], m.Version)
	binary.LittleEndian.PutUint64(buff[20:28], uint64(m.BuildTime.Unix()))
	return buff, nil
}

func (m *FileHeader) UnmarshalBinary(data []byte) error {
	if !bytes.Equal(data[:16], m.Marker[:]) {
		return ErrCorrupted
	}
	// copy(m.Marker[:], data[:16])
	m.Version = binary.LittleEndian.Uint32(data[16:20])
	m.BuildTime = time.Unix(int64(binary.LittleEndian.Uint64(data[20:28])), 0)
	return nil
}

func (m *FileHeader) readHeaderFromFile(file *os.File) error {
	buff := make([]byte, fileHeaderSize)
	if _, err := file.Read(buff); err != nil {
		return err
	}
	return m.UnmarshalBinary(buff)
}

func (m *FileHeader) writeHeaderToFile(file *os.File) error {
	buff, err := m.MarshalBinary()
	if err != nil {
		return err
	}
	if _, err := file.Write(buff); err != nil {
		return err
	}
	return m.UnmarshalBinary(buff)
}

func (m *FileHeader) checkVersion() bool {
	return m.Version == CurrentDatabaseVersion
}

func NewFileHeader(marker [16]byte) *FileHeader {
	return &FileHeader{
		Marker:    marker,
		Version:   CurrentDatabaseVersion,
		BuildTime: time.Now(),
	}
}
