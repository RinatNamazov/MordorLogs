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
	"encoding/binary"
	"os"
)

/* Second index file: Offsets to data file.
============================SecondIndexEntry===========================
	OffsetCount			= 4 byte
	Offset				= 8 byte // repeated
============================SecondIndexEntry===========================
*/

var secondIndexHeaderMarker = [16]byte{'M', 'o', 'r', 'd', 'o', 'r', 'L', 'o', 'g', 's', 'D', 'B', 0x02, 0x02, 0x02, 0x02}

type SecondIndexFile struct {
	file        *os.File
	writeOffset int64
}

func (m *SecondIndexFile) Open(filePath string) (isnew bool, err error) {
	m.file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return false, err
	}
	stat, err := m.file.Stat()
	if err != nil {
		return false, err
	}
	fileSize := stat.Size()
	if fileSize == 0 {
		if err := m.writeHeader(); err != nil {
			return true, err
		}
		m.writeOffset = fileHeaderSize
		return true, nil
	} else {
		if err := m.readHeader(); err != nil {
			return false, err
		}
		m.writeOffset = fileSize
	}
	return false, err
}

func (m *SecondIndexFile) Close() error {
	return m.file.Close()
}

func (m *SecondIndexFile) Sync() error {
	return m.file.Sync()
}

func (m *SecondIndexFile) readHeader() error {
	fh := NewFileHeader(secondIndexHeaderMarker)
	if err := fh.readHeaderFromFile(m.file); err != nil {
		return err
	}
	if !fh.checkVersion() {
		return ErrIncompatibleVersions
	}
	return nil
}

func (m *SecondIndexFile) writeHeader() error {
	fh := NewFileHeader(secondIndexHeaderMarker)
	if err := fh.writeHeaderToFile(m.file); err != nil {
		return err
	}
	return nil
}

func (m *SecondIndexFile) WriteEntry(offsets []uint64) (uint64, error) {
	c := len(offsets)
	if c == 0 {
		return 0, ErrEmptySlice
	}
	returnOffset := uint64(m.writeOffset - fileHeaderSize)

	bc := make([]byte, 4)
	binary.LittleEndian.PutUint32(bc, uint32(c))
	if _, err := m.file.WriteAt(bc, m.writeOffset); err != nil {
		return 0, err
	}
	m.writeOffset += 4

	b := make([]byte, 8)
	for _, offset := range offsets {
		binary.LittleEndian.PutUint64(b, offset)
		if _, err := m.file.WriteAt(b, m.writeOffset); err != nil {
			return 0, err
		}
		m.writeOffset += 8
	}

	return returnOffset, nil
}

func (m *SecondIndexFile) ReadEntryAt(offset uint64) ([]uint64, error) {
	offset += uint64(fileHeaderSize)

	bc := make([]byte, 4)
	if _, err := m.file.ReadAt(bc, int64(offset)); err != nil {
		return nil, err
	}
	count := uint64(binary.LittleEndian.Uint32(bc))
	offsets := make([]uint64, count)
	b := make([]byte, 8)

	offset += 4
	for i := range offsets {
		if _, err := m.file.ReadAt(b, int64(offset)); err != nil {
			return nil, err
		}
		offsets[i] = binary.LittleEndian.Uint64(b)
		offset += 8
	}

	return offsets, nil
}

func (m *SecondIndexFile) Iterator() *SecondIndexIterator {
	return &SecondIndexIterator{file: m.file, fileSize: m.writeOffset, offset: fileHeaderSize}
}
