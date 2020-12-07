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
	"os"
	"path/filepath"
)

type MordorLogsDB struct {
	firstIndex  FirstIndexFile
	secondIndex SecondIndexFile
	data        DataFile
}

func (m *MordorLogsDB) Open(dirPath string) (isnew bool, err error) {
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		return false, err
	}

	var firstIndexIsNew, secondIndexIsNew, dataIsNew bool

	if firstIndexIsNew, err = m.firstIndex.Open(filepath.Join(dirPath, "first_index.bin")); err != nil {
		return false, err
	}
	if secondIndexIsNew, err = m.secondIndex.Open(filepath.Join(dirPath, "second_index.bin")); err != nil {
		m.firstIndex.Close()
		return false, err
	}
	if dataIsNew, err = m.data.Open(filepath.Join(dirPath, "data.bin")); err != nil {
		m.firstIndex.Close()
		m.secondIndex.Close()
		return false, err
	}

	n := 0
	if firstIndexIsNew {
		n++
	}
	if secondIndexIsNew {
		n++
	}
	if dataIsNew {
		n++
	}
	allFilesIsNew := n == 3
	if !allFilesIsNew && n != 0 {
		// Some files have just been created and some have already been created.
		return false, ErrCorrupted
	}

	return allFilesIsNew, nil
}

func (m *MordorLogsDB) Close() error {
	if err := m.data.Close(); err != nil {
		return err
	}
	if err := m.secondIndex.Close(); err != nil {
		return err
	}
	if err := m.firstIndex.Close(); err != nil {
		return err
	}
	return nil
}

func (m *MordorLogsDB) SyncFiles() error {
	if err := m.firstIndex.Sync(); err != nil {
		return err
	}
	if err := m.secondIndex.Sync(); err != nil {
		return err
	}
	if err := m.data.Sync(); err != nil {
		return err
	}
	return nil
}

func (m *MordorLogsDB) GetEntryCount() int {
	return m.firstIndex.GetEntryCount()
}

func (m *MordorLogsDB) Write(nickname string, data DataEntry) error {
	entrys := []*DataEntry{&data}
	return m.WriteAll(nickname, entrys)
}

func (m *MordorLogsDB) WriteAll(nickname string, entrys []*DataEntry) error {
	if len(nickname) > 24 {
		return ErrLongNickName
	}
	entrysLen := len(entrys)
	if entrysLen == 0 {
		return ErrEmptySlice
	}
	// Check everything before writing.
	for _, data := range entrys {
		if data == nil {
			return ErrNullPointer
		}
		if err := data.Validate(); err != nil {
			return err
		}
	}

	offsetsToData := make([]uint64, entrysLen)
	for i, data := range entrys {
		dataOffset, err := m.data.WriteEntry(*data)
		if err != nil {
			return err
		}
		offsetsToData[i] = dataOffset
	}

	offsetToSecondIndex, err := m.secondIndex.WriteEntry(offsetsToData)
	if err != nil {
		return err
	}

	_, err = m.firstIndex.WriteEntry(nickname, offsetToSecondIndex)
	if err != nil {
		return err
	}

	return nil
}

func (m *MordorLogsDB) readDataBySecondIndex(offsetToSecondIndex uint64) ([]*DataEntry, error) {
	offsetsToData, err := m.secondIndex.ReadEntryAt(offsetToSecondIndex)
	if err != nil {
		return nil, err
	}

	entrys := make([]*DataEntry, len(offsetsToData))
	for i, offsetToData := range offsetsToData {
		data, err := m.data.ReadEntryAt(offsetToData)
		if err != nil {
			return nil, err
		}
		entrys[i] = data
	}

	return entrys, nil
}

// It is used when the data has not been sorted, which makes it impossible to apply binary search.
func (m *MordorLogsDB) NoBinaryFindDataByNickName(nickname string) ([]*DataEntry, error) {
	offsetToSecondIndex, err := m.firstIndex.NoBinaryFindOffsetByNickName(nickname)
	if err != nil {
		return nil, err
	}
	return m.readDataBySecondIndex(offsetToSecondIndex)
}

func (m *MordorLogsDB) FindDataByNickName(nickname string) ([]*DataEntry, error) {
	offsetToSecondIndex, err := m.firstIndex.FindOffsetByNickName(nickname)
	if err != nil {
		return nil, err
	}
	return m.readDataBySecondIndex(offsetToSecondIndex)
}

// Iterates over all elements. Used in the early stages of converting logs to a database.
func (m *MordorLogsDB) FindAllDataByNickName(nickname string) ([]*DataEntry, error) {
	offsetsToSecondIndex, err := m.firstIndex.FindAllOffsetsByNickName(nickname)
	if err != nil {
		return nil, err
	}

	allentrys := make([]*DataEntry, 0)
	for _, offsetToSecondIndex := range offsetsToSecondIndex {
		offsetsToData, err := m.secondIndex.ReadEntryAt(offsetToSecondIndex)
		if err != nil {
			return nil, err
		}

		entrys := make([]*DataEntry, len(offsetsToData))
		for i, offsetToData := range offsetsToData {
			data, err := m.data.ReadEntryAt(offsetToData)
			if err != nil {
				return nil, err
			}
			entrys[i] = data
		}
		allentrys = append(allentrys, entrys...)
	}

	return allentrys, nil
}

func (m *MordorLogsDB) ReadDataEntryAt(offset uint64) (*DataEntry, error) {
	return m.data.ReadEntryAt(offset)
}

func (m *MordorLogsDB) ReadSecondIndexEntryAt(offset uint64) ([]uint64, error) {
	return m.secondIndex.ReadEntryAt(offset)
}

func (m *MordorLogsDB) Iterator() *DataIterator {
	return &DataIterator{firstIndexIterator: m.firstIndex.Iterator(), db: m}
}

func (m *MordorLogsDB) FirstIndexIterator() *FirstIndexIterator {
	return m.firstIndex.Iterator()
}

func (m *MordorLogsDB) SecondIndexIterator() *SecondIndexIterator {
	return m.secondIndex.Iterator()
}

func NewMordorLogsDB(dirPath string) (*MordorLogsDB, bool, error) {
	mldb := new(MordorLogsDB)
	isnew, err := mldb.Open(dirPath)
	if err != nil {
		return nil, isnew, err
	}
	return mldb, isnew, nil
}
