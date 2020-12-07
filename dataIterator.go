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

type DataIterator struct {
	firstIndexIterator *FirstIndexIterator
	db                 *MordorLogsDB

	// Used in the early stages of converting logs to a database.
	data          map[string][]uint64
	usedNickNames map[string]bool
}

func (m *DataIterator) Next() (string, []*DataEntry, error) {
	if m.usedNickNames == nil {
		m.usedNickNames = make(map[string]bool)
	}
	var nickname string
	var err error
	for {
		nickname, _, err = m.firstIndexIterator.Next()
		if err == ErrIterationDone {
			m.usedNickNames = nil // clear
			break
		} else if err != nil {
			return "", nil, err
		}

		if _, ok := m.usedNickNames[nickname]; !ok {
			m.usedNickNames[nickname] = true
			break
		}
	}

	data, err := m.db.FindAllDataByNickName(nickname)
	if err != nil {
		return "", nil, err
	}
	return nickname, data, nil
}

func (m *DataIterator) ReadToMemory() error {
	m.data = make(map[string][]uint64)
	for {
		nickname, offsetToSecondIndex, err := m.firstIndexIterator.Next()
		if err == ErrIterationDone {
			break
		} else if err != nil {
			return err
		}

		offsetsToData, err := m.db.ReadSecondIndexEntryAt(offsetToSecondIndex)
		if err != nil {
			return err
		}
		offsetToData := offsetsToData[0]

		if offsets, ok := m.data[nickname]; ok {
			offsets = append(offsets, offsetToData)
			m.data[nickname] = offsets
		} else {
			offsets := make([]uint64, 1)
			offsets[0] = offsetToData
			m.data[nickname] = offsets
		}
	}

	return nil
}

func (m *DataIterator) GetMap() map[string][]uint64 {
	return m.data
}
