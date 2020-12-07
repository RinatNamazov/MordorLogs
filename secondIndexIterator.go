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

type SecondIndexIterator struct {
	file     *os.File
	fileSize int64
	offset   int64
}

func (m *SecondIndexIterator) Next() ([]uint64, error) {
	if m.offset < m.fileSize {
		bc := make([]byte, 4)
		if _, err := m.file.ReadAt(bc, m.offset); err != nil {
			return nil, err
		}
		count := uint64(binary.LittleEndian.Uint32(bc))
		offsets := make([]uint64, count)
		b := make([]byte, 8)

		m.offset += 4
		for i := range offsets {
			if _, err := m.file.ReadAt(b, m.offset); err != nil {
				return nil, err
			}
			offsets[i] = binary.LittleEndian.Uint64(b)
			m.offset += 8
		}
		return offsets, nil
	}
	return nil, ErrIterationDone
}
