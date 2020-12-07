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

type FirstIndexIterator struct {
	file     *os.File
	fileSize int64
	offset   int64
}

func (m *FirstIndexIterator) Next() (string, uint64, error) {
	entry := make([]byte, firstIndexEntrySize)

	if m.offset < m.fileSize {
		if _, err := m.file.ReadAt(entry, int64(m.offset)); err != nil {
			return "", 0, err
		}
		m.offset += firstIndexEntrySize

		nick := entry[:24]
		if zeroIndex := bytes.IndexByte(nick, 0x00); zeroIndex != -1 {
			nick = nick[:zeroIndex]
		}

		return string(nick), binary.LittleEndian.Uint64(entry[24:32]), nil
	}

	return "", 0, ErrIterationDone
}
