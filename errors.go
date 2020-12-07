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

import "errors"

var ErrCorrupted = errors.New("database is corrupted")
var ErrIncompatibleVersions = errors.New("database has incompatible versions")
var ErrEmptySlice = errors.New("empty slice")
var ErrEntryNotFound = errors.New("entry not found")
var ErrLongNickName = errors.New("nickname too long")
var ErrLongIP = errors.New("only IPv4 address allowed")
var ErrLongStr = errors.New("max string length 255 characters")
var ErrIterationDone = errors.New("no more items in iterator")
var ErrNullPointer = errors.New("null pointer")
