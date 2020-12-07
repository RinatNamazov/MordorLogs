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
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const TG_BOT_API = ""

const helpMessage = "Привет, отправь мне ник игрока с Mordor RP.\nНикнейм может включать только следующие символы: `a-z`, `A-Z`, `0-9`, `[]`, `()`, `$`, `@`, `.`, `_`, `=`, а длина должна быть не менее 3 символов и не более 24."

const timeFormatLayout = "02.01.2006 15:04:05"

var validNickName = regexp.MustCompile(`^[a-zA-Z0-9\[\]\(\)\$@\._=]{3,24}$`)

var (
	bot  *tgbotapi.BotAPI
	mldb *MordorLogsDB
)

func main() {
	var err error

	mldb, _, err = NewMordorLogsDB("./mordor.db")
	if err != nil {
		log.Panicln(err)
	}
	defer mldb.Close()

	fmt.Println("Number of nicknames:", mldb.GetEntryCount())

	bot, err = tgbotapi.NewBotAPI(TG_BOT_API)
	if err != nil {
		log.Panicln(err)
	}

	log.Println("Authorized on account:", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panicln(err)
	}

	for update := range updates {
		if update.Message != nil {
			handleMessage(update.Message)
		}
	}
}

func handleFindDataErrors(err error) string {
	if err == ErrLongNickName {
		return "Максимальная длина ника 24 символа."
	} else if err == ErrEntryNotFound {
		return "Игрок с данным ником не найден в базе."
	} else if err != nil {
		log.Println(err)
		return "Произошла внутренняя ошибка, сообщите об этом создателю бота."
	}
	return ""
}

func sendMarkDownMessage(chatId int64, message string) {
	msg := tgbotapi.NewMessage(chatId, message)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := bot.Send(msg); err != nil {
		fmt.Println(err)
	}
}

func formatLogData(nickname string, data *DataEntry) string {
	return fmt.Sprintf("*NickName:* %s\n*Time:* %s\n*IP:* `%s`\n*Android:* %s\n*Brand:* %s\n*Model:* %s\n*Fingerprint:* %s\n*Server:* `%s`",
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, nickname),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Time.Format(timeFormatLayout)),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.IP.String()),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Android),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Brand),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Model),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Fingerprint),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, data.Server))
}

func handleMessage(msg *tgbotapi.Message) {
	splitText := strings.Split(msg.Text, " ")
	splitCount := len(splitText)
	if splitCount == 1 {
		nickname := splitText[0]
		if !validNickName.MatchString(nickname) {
			sendMarkDownMessage(msg.Chat.ID, helpMessage)
			return
		}

		entrys, err := mldb.FindDataByNickName(nickname)
		if errmsg := handleFindDataErrors(err); errmsg != "" {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errmsg))
			return
		}
		entrysCount := len(entrys)

		if entrysCount == 1 {
			sendMarkDownMessage(msg.Chat.ID, formatLogData(nickname, entrys[0]))
		} else if entrysCount <= 20 {
			output := fmt.Sprintf("Найдено %d записей:\n\n", entrysCount)
			for i, data := range entrys {
				output += fmt.Sprintf("%d: %s\n", i+1, data.Time.Format(timeFormatLayout))
			}
			output += fmt.Sprintf("\nВведите `%s id`.", nickname)

			sendMarkDownMessage(msg.Chat.ID, output)
		} else {
			sendMarkDownMessage(msg.Chat.ID, fmt.Sprintf("Найдено %d записей. Введите `%s ID записи`.", entrysCount, nickname))
		}
	} else if splitCount == 2 {
		nickname := splitText[0]
		id, err := strconv.ParseUint(splitText[1], 10, 32)
		if err != nil || id == 0 {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Необходимо вводить положительный целочисленный ID записи."))
			return
		}

		entrys, err := mldb.FindDataByNickName(nickname)
		if errmsg := handleFindDataErrors(err); errmsg != "" {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, errmsg))
			return
		}

		entrysCount := len(entrys)
		if int(id) > entrysCount {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("ID записи для ника %s не должно превышать %d.", nickname, entrysCount)))
			return
		}
		data := entrys[id-1]
		sendMarkDownMessage(msg.Chat.ID, formatLogData(nickname, data))
	} else {
		sendMarkDownMessage(msg.Chat.ID, helpMessage)
	}
}
