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
)

/* First index file: Offset to second index file.
Calculate count of entrys: (fileSize - fileHeaderSize) / firstIndexEntrySize
============================FisrtIndexEntry===========================
	NickName			= 24 bytes
	Offset				= 8 byte
============================FisrtIndexEntry===========================
*/

const firstIndexEntrySize = 32

var firstIndexHeaderMarker = [16]byte{'M', 'o', 'r', 'd', 'o', 'r', 'L', 'o', 'g', 's', 'D', 'B', 0x01, 0x01, 0x01, 0x01}

type FirstIndexFile struct {
	file        *os.File
	writeOffset int64
	entryCount  int
}

func (m *FirstIndexFile) Open(filePath string) (isnew bool, err error) {
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
		m.entryCount = int((fileSize - fileHeaderSize) / firstIndexEntrySize)
		m.writeOffset = fileSize
	}
	return false, err
}

func (m *FirstIndexFile) Close() error {
	return m.file.Close()
}

func (m *FirstIndexFile) Sync() error {
	return m.file.Sync()
}

func (m *FirstIndexFile) readHeader() error {
	fh := NewFileHeader(firstIndexHeaderMarker)
	if err := fh.readHeaderFromFile(m.file); err != nil {
		return err
	}
	if !fh.checkVersion() {
		return ErrIncompatibleVersions
	}
	return nil
}

func (m *FirstIndexFile) writeHeader() error {
	fh := NewFileHeader(firstIndexHeaderMarker)
	if err := fh.writeHeaderToFile(m.file); err != nil {
		return err
	}
	return nil
}

func (m *FirstIndexFile) GetEntryCount() int {
	return m.entryCount
}

func (m *FirstIndexFile) WriteEntry(nickname string, offset uint64) (uint64, error) {
	if len(nickname) > 24 {
		return 0, ErrLongNickName
	}
	nick := make([]byte, 24)
	copy(nick, nickname)
	if _, err := m.file.WriteAt(nick, m.writeOffset); err != nil {
		return 0, err
	}
	m.writeOffset += 24

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, offset)
	if _, err := m.file.WriteAt(b, m.writeOffset); err != nil {
		return 0, err
	}
	m.writeOffset += 8

	m.entryCount++

	return uint64(m.writeOffset - fileHeaderSize), nil
}

// It is used when the data has not been sorted, which makes it impossible to apply binary search.
func (m *FirstIndexFile) NoBinaryFindOffsetByNickName(nickname string) (uint64, error) {
	nickLength := len(nickname)
	if nickLength > 24 {
		return 0, ErrLongNickName
	}
	nick := []byte(nickname)
	entry := make([]byte, firstIndexEntrySize)

	for offset := fileHeaderSize; offset < m.writeOffset; offset += firstIndexEntrySize {
		if _, err := m.file.ReadAt(entry, offset); err != nil {
			return 0, err
		}
		// Exact match only.
		if bytes.Index(entry[:24], nick) == 0 && entry[nickLength+1] == 0x00 {
			return binary.LittleEndian.Uint64(entry[24:32]), nil
		}
	}

	return 0, ErrEntryNotFound
}

func (m *FirstIndexFile) FindOffsetByNickName(nickname string) (uint64, error) {
	nickLength := len(nickname)
	if nickLength > 24 {
		return 0, ErrLongNickName
	}
	entry := make([]byte, firstIndexEntrySize)

	left := 0
	right := m.entryCount - 1
	for left <= right {
		mid := (left + right) / 2

		offset := fileHeaderSize + (int64(mid) * firstIndexEntrySize)
		if _, err := m.file.ReadAt(entry, offset); err != nil {
			return 0, err
		}

		entryNick := entry[:24]
		if zeroIndex := bytes.IndexByte(entryNick, 0x00); zeroIndex != -1 {
			entryNick = entryNick[:zeroIndex]
		}
		entryNickName := string(entryNick)

		if nickname > entryNickName {
			left = mid + 1
		} else if nickname < entryNickName {
			right = mid - 1
		} else { // Match found.
			return binary.LittleEndian.Uint64(entry[24:32]), nil
		}
	}

	return 0, ErrEntryNotFound
}

// Iterates over all elements. Used in the early stages of converting logs to a database.
func (m *FirstIndexFile) FindAllOffsetsByNickName(nickname string) ([]uint64, error) {
	nickLength := len(nickname)
	if nickLength > 24 {
		return nil, ErrLongNickName
	}
	nick := []byte(nickname)
	entry := make([]byte, firstIndexEntrySize)

	offsets := make([]uint64, 0)

	for offset := fileHeaderSize; offset < m.writeOffset; offset += firstIndexEntrySize {
		if _, err := m.file.ReadAt(entry, offset); err != nil {
			return nil, err
		}
		// Exact match only.
		if bytes.Index(entry[:24], nick) == 0 && entry[nickLength+1] == 0x00 {
			offsets = append(offsets, binary.LittleEndian.Uint64(entry[24:32]))
		}
	}

	if len(offsets) == 0 {
		return nil, ErrEntryNotFound
	}

	return offsets, nil
}

func (m *FirstIndexFile) Iterator() *FirstIndexIterator {
	return &FirstIndexIterator{file: m.file, fileSize: m.writeOffset, offset: fileHeaderSize}
}
