package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/a-kataev/tg"
	"golang.org/x/exp/slog"
)

func main() {
	fset := flag.NewFlagSet("", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	token := fset.String("token", "", "token")
	chatID := fset.Int64("chat_id", 0, "chat id")
	text := fset.String("text", "", "text")
	parseMode := fset.String("parse_mode", "Markdown", "parse mode")

	log := slog.New(slog.NewJSONHandler(os.Stdout))

	logFatal := func(msg string) {
		log.Error(msg)

		os.Exit(1)
	}

	err := fset.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(flag.ErrHelp, err) {
			fmt.Fprintln(os.Stderr, "tg - telegram bot 🤖 send message ✉️")
			fset.SetOutput(os.Stderr)
			fset.Usage()

			os.Exit(1)
		}

		logFatal(err.Error())
	}

	if *token == "" {
		*token = os.Getenv("TOKEN")
	}

	if *token == "" {
		logFatal("empty token")
	}

	if *chatID < 1 {
		logFatal("invalid chat id")
	}

	if *text == "" {
		logFatal("empty text")
	}

	if *text == "-" {
		const maxTextSize int64 = 4096

		stdin, err := io.ReadAll(io.LimitReader(os.Stdin, maxTextSize))
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logFatal(err.Error())
			}
		}

		*text = string(stdin)
	}

	tg, err := tg.NewTG(
		*token,
		tg.MessageParseMode(tg.ParseMode(*parseMode)),
	)
	if err != nil {
		logFatal(err.Error())
	}

	ctx := context.Background()

	bot, err := tg.GetMe(ctx)
	if err != nil {
		logFatal(err.Error())
	}

	log = log.With(slog.String("bot_name", bot.UserName))

	msg, err := tg.SendMessage(ctx, *chatID, *text)
	if err != nil {
		logFatal(err.Error())
	}

	log.Info("Success send message",
		slog.Int64("chat_id", *chatID),
		slog.Int("message_id", msg.MessageID),
	)
}
