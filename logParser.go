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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ConvertLogsToDatabase => MemSortDatabase => SortFirstIndex => SortSecondIndex

type firstIndexItem struct {
	NickName string
	Offset   uint64
}

func SortFirstIndex(from *MordorLogsDB, firstIndexFile string) error {
	var firstIndex FirstIndexFile
	if _, err := firstIndex.Open(firstIndexFile); err != nil {
		return err
	}

	items := make([]firstIndexItem, 0, from.GetEntryCount())
	it := from.FirstIndexIterator()
	for {
		nickname, offset, err := it.Next()
		if err == ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		items = append(items, firstIndexItem{nickname, offset})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].NickName < items[j].NickName })

	for _, v := range items {
		fmt.Println(v.NickName, v.Offset)
		if _, err := firstIndex.WriteEntry(v.NickName, v.Offset); err != nil {
			return err
		}
	}

	return nil
}

type secondIndexItem struct {
	Time   time.Time
	Offset uint64
}

func SortSecondIndex(from *MordorLogsDB, secondIndexFile string) error {
	var secondIndex SecondIndexFile
	if _, err := secondIndex.Open(secondIndexFile); err != nil {
		return err
	}

	it := from.SecondIndexIterator()
	for {
		offsetsToData, err := it.Next()
		if err == ErrIterationDone {
			break
		} else if err != nil {
			return err
		}

		items := make([]secondIndexItem, len(offsetsToData))
		for i, v := range offsetsToData {
			data, err := from.ReadDataEntryAt(v)
			if err != nil {
				return err
			}
			items[i] = secondIndexItem{data.Time, v}
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })

		offsets := make([]uint64, len(items))
		for i, v := range items {
			offsets[i] = v.Offset
		}
		secondIndex.WriteEntry(offsets)
	}

	return nil
}

// Very slow, don't use.
func SortDatabase(from *MordorLogsDB, to *MordorLogsDB) error {
	it := from.Iterator()
	for {
		nickname, entrys, err := it.Next()
		if err == ErrIterationDone {
			break
		} else if err != nil {
			return err
		}
		fmt.Println(nickname, len(entrys))
		if err := to.WriteAll(nickname, entrys); err != nil {
			return fmt.Errorf("to.WriteAll failed: %w", err)
		}
	}
	return nil
}

func MemSortDatabase(from *MordorLogsDB, to *MordorLogsDB) error {
	it := from.Iterator()
	if err := it.ReadToMemory(); err != nil {
		return err
	}
	fmt.Println("FirstIndex read into memory.")
	data := it.GetMap()
	for nickname, offsets := range data {
		offsetsCount := len(offsets)
		fmt.Println(nickname, offsetsCount)
		entrys := make([]*DataEntry, offsetsCount)

		for i, offset := range offsets {
			entry, err := from.ReadDataEntryAt(offset)
			if err != nil {
				return err
			}
			entrys[i] = entry
		}

		if err := to.WriteAll(nickname, entrys); err != nil {
			return fmt.Errorf("to.WriteAll failed: %w", err)
		}
	}
	return nil
}

func ConvertLogsToDatabase(dirPath string, mldb *MordorLogsDB) error {
	return parseDateDirLogs(filepath.Join(dirPath, "client_log"), mldb)
}

func parseDateDirLogs(dirPath string, mldb *MordorLogsDB) error {
	dateDirs, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, file := range dateDirs {
		if file.IsDir() {
			if err := parseDirLogs(filepath.Join(dirPath, file.Name()), mldb); err != nil {
				log.Println("parseDirLogs:", err)
			}
		}
	}
	return nil
}

func parseDirLogs(dirPath string, mldb *MordorLogsDB) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(info.Name()) == ".log" {
			nickName := info.Name()
			nickName = nickName[:len(nickName)-4] // len(".log") == 4
			if nickName == "today" {
				return nil
			}
			items, err := ParseLogFile(path)
			if err != nil {
				return fmt.Errorf("ParseLogFile failed: %w", err)
			}

			for _, data := range items {
				if err := mldb.Write(nickName, data); err != nil {
					return fmt.Errorf("db.Write failed: %w", err)
				}

				/*
					fmt.Println("")
					fmt.Println("NickName:", nickName)
					fmt.Println("Time:", v.Time.Format("02.01.2006 15:04:05"))
					fmt.Println("Android:", v.Android)
					fmt.Println("Brand:", v.Brand)
					fmt.Println("Model:", v.Model)
					fmt.Println("Fingerprint:", v.Fingerprint)
					fmt.Println("IP:", v.IP)
					fmt.Println("Server:", v.Server)
					fmt.Println("")
				*/
			}
		}
		return nil
	})
}

func ParseLogFile(filePath string) ([]DataEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	logLines := make([]DataEntry, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// >> [01.08.2020 21:04:48] Android: 10 | Brand: Xiaomi | Model: Mi 10 | FP: test | IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line := scanner.Bytes()

		// len(">> [01.08.2020 21:04:48] Android:  | Brand:  | Model:  | FP:  | IP:  | Server: ") == 79
		//      01 3                  23
		if len(line) <= 79 {
			continue
		}
		if line[0] != '>' || line[1] != '>' {
			continue
		}
		//if line[3] != '[' || line[23] != ']' {
		//	continue
		//}

		var e DataEntry

		e.Time, err = time.Parse("02.01.2006 15:04:05", string(line[4:23]))
		if err != nil {
			log.Println("Failed parse time:", string(line[4:23]), err)
			continue
		}

		// Android: 10 | Brand: Xiaomi | Model: Mi 10 | FP: test | IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line = line[25:]
		//if !bytes.Equal(line[:8], []byte("Android:")) {
		//	continue
		//}
		separatorIndex := bytes.IndexRune(line, '|')
		e.Android = string(line[9 : separatorIndex-1])

		// Brand: Xiaomi | Model: Mi 10 | FP: test | IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line = line[separatorIndex+2:]
		//if !bytes.Equal(line[:6], []byte("Brand:")) {
		//	continue
		//}
		separatorIndex = bytes.IndexRune(line, '|')
		e.Brand = string(line[7 : separatorIndex-1])

		// Model: Mi 10 | FP: test | IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line = line[separatorIndex+2:]
		//if !bytes.Equal(line[:6], []byte("Model:")) {
		//	continue
		//}
		separatorIndex = bytes.IndexRune(line, '|')
		e.Model = string(line[7 : separatorIndex-1])

		// FP: test | IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line = line[separatorIndex+2:]
		//if !bytes.Equal(line[:3], []byte("FP:")) {
		//	continue
		//}
		separatorIndex = bytes.IndexRune(line, '|')
		e.Fingerprint = string(line[4 : separatorIndex-1])

		// IP: 192.168.1.1 | Server: 192.168.1.1:7777
		line = line[separatorIndex+2:]
		//if !bytes.Equal(line[:3], []byte("IP:")) {
		//	continue
		//}
		separatorIndex = bytes.IndexRune(line, '|')
		e.IP = net.ParseIP(string(line[4 : separatorIndex-1]))
		if e.IP == nil {
			log.Println("Failed parse IP address:", string(line[4:separatorIndex-1]))
			continue
		}

		// Server: 192.168.1.1:7777
		line = line[separatorIndex+2:]
		//if !bytes.Equal(line[:7], []byte("Server:")) {
		//	continue
		//}
		e.Server = string(line[8:])

		logLines = append(logLines, e)
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	return logLines, nil
}
