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
	"net"
	"os"
	"time"
)

/* Data file: Contains information about player.
===============================DataEntry==============================
	Time				= 8 byte
	IP					= 4 byte
	AndroidLength		= 1 byte
	Android				= AndroidLength byte
	BrandLength			= 1 byte
	Brand				= BrandLength byte
	ModelLength			= 1 byte
	Model				= ModelLength byte
	FingerprintLength	= 1 byte
	Fingerprint			= FingerprintLength byte
	ServerLength		= 1 byte
	Server				= ServerLength byte
===============================DataEntry==============================
*/

type DataEntry struct {
	Time        time.Time
	IP          net.IP
	Android     string
	Brand       string
	Model       string
	Fingerprint string
	Server      string
}

func (m *DataEntry) Validate() error {
	if len(m.IP.To4()) > 4 {
		return ErrLongIP
	}
	if len(m.Android) > 255 || len(m.Brand) > 255 || len(m.Model) > 255 || len(m.Fingerprint) > 255 || len(m.Server) > 255 {
		return ErrLongStr
	}
	return nil
}

var dataHeaderMarker = [16]byte{'M', 'o', 'r', 'd', 'o', 'r', 'L', 'o', 'g', 's', 'D', 'B', 0x03, 0x03, 0x03, 0x03}

type DataFile struct {
	file        *os.File
	writeOffset int64
}

func (m *DataFile) Open(filePath string) (isnew bool, err error) {
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

func (m *DataFile) Close() error {
	return m.file.Close()
}

func (m *DataFile) Sync() error {
	return m.file.Sync()
}

func (m *DataFile) readHeader() error {
	fh := NewFileHeader(dataHeaderMarker)
	if err := fh.readHeaderFromFile(m.file); err != nil {
		return err
	}
	if !fh.checkVersion() {
		return ErrIncompatibleVersions
	}
	return nil
}

func (m *DataFile) writeHeader() error {
	fh := NewFileHeader(dataHeaderMarker)
	if err := fh.writeHeaderToFile(m.file); err != nil {
		return err
	}
	return nil
}

func (m *DataFile) writeString8(str string) error {
	length := []byte{0}
	length[0] = uint8(len(str))
	if _, err := m.file.WriteAt(length, m.writeOffset); err != nil {
		return err
	}
	m.writeOffset++
	strBytes := []byte(str)
	if _, err := m.file.WriteAt(strBytes, m.writeOffset); err != nil {
		return err
	}
	m.writeOffset += int64(length[0])
	return nil
}

func (m *DataFile) WriteEntry(entry DataEntry) (uint64, error) {
	if err := entry.Validate(); err != nil {
		return 0, err
	}
	returnOffset := uint64(m.writeOffset - fileHeaderSize)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(entry.Time.Unix()))
	if _, err := m.file.WriteAt(b, m.writeOffset); err != nil {
		return 0, err
	}
	m.writeOffset += 8

	if _, err := m.file.WriteAt(entry.IP.To4(), m.writeOffset); err != nil {
		return 0, err
	}
	m.writeOffset += 4

	if err := m.writeString8(entry.Android); err != nil {
		return 0, err
	}
	if err := m.writeString8(entry.Brand); err != nil {
		return 0, err
	}
	if err := m.writeString8(entry.Model); err != nil {
		return 0, err
	}
	if err := m.writeString8(entry.Fingerprint); err != nil {
		return 0, err
	}
	if err := m.writeString8(entry.Server); err != nil {
		return 0, err
	}

	return returnOffset, nil
}

func (m *DataFile) ReadEntryAt(off uint64) (*DataEntry, error) {
	offset := int64(off) + fileHeaderSize

	b := make([]byte, 8)
	if _, err := m.file.ReadAt(b, offset); err != nil {
		return nil, err
	}

	offset += 8
	entry := new(DataEntry)
	entry.Time = time.Unix(int64(binary.LittleEndian.Uint64(b)), 0)

	bip := make([]byte, 4)
	if _, err := m.file.ReadAt(bip, offset); err != nil {
		return nil, err
	}
	offset += 4
	entry.IP = net.IP(bip)

	length := make([]byte, 1)
	if _, err := m.file.ReadAt(length, offset); err != nil {
		return nil, err
	}
	offset++
	android := make([]byte, length[0])
	if _, err := m.file.ReadAt(android, offset); err != nil {
		return nil, err
	}
	offset += int64(length[0])
	entry.Android = string(android)

	if _, err := m.file.ReadAt(length, offset); err != nil {
		return nil, err
	}
	offset++
	brand := make([]byte, length[0])
	if _, err := m.file.ReadAt(brand, offset); err != nil {
		return nil, err
	}
	offset += int64(length[0])
	entry.Brand = string(brand)

	if _, err := m.file.ReadAt(length, offset); err != nil {
		return nil, err
	}
	offset++
	model := make([]byte, length[0])
	if _, err := m.file.ReadAt(model, offset); err != nil {
		return nil, err
	}
	offset += int64(length[0])
	entry.Model = string(model)

	if _, err := m.file.ReadAt(length, offset); err != nil {
		return nil, err
	}
	offset++
	fingerprint := make([]byte, length[0])
	if _, err := m.file.ReadAt(fingerprint, offset); err != nil {
		return nil, err
	}
	offset += int64(length[0])
	entry.Fingerprint = string(fingerprint)

	if _, err := m.file.ReadAt(length, offset); err != nil {
		return nil, err
	}
	offset++
	server := make([]byte, length[0])
	if _, err := m.file.ReadAt(server, offset); err != nil {
		return nil, err
	}
	// offset += int64(length[0])
	entry.Server = string(server)

	return entry, nil
}
