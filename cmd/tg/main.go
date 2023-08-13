package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/a-kataev/tg"
)

func main() {
	fset := flag.NewFlagSet("", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	token := fset.String("token", "", "token (environment TG_TOKEN)")
	chatID := fset.Int64("chat_id", 0, "chat id")
	text := fset.String("text", "", "text (use - for read pipe)")
	parseMode := fset.String("parse_mode", "Markdown", "parse mode")
	messageThreadID := fset.Int64("message_thread_id", 0, "message thread id")
	disableWebPagePreview := fset.Bool("disable_web_page_preview", false, "disable web page preview")
	disableNotification := fset.Bool("disable_notification", false, "disable notification")
	protectContent := fset.Bool("protect_content", false, "protect content")

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

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
		*token = os.Getenv("TG_TOKEN")
	}

	if *text == "-" {
		stdin, err := io.ReadAll(io.LimitReader(os.Stdin, int64(tg.MaxTextSize)))
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logFatal(err.Error())
			}
		}

		*text = string(stdin)
	}

	tgb, err := tg.NewTG(*token)
	if err != nil {
		logFatal(err.Error())
	}

	ctx := context.Background()

	bot, err := tgb.GetMe(ctx)
	if err != nil {
		logFatal(err.Error())
	}

	log = log.With(slog.String("bot_name", bot.UserName))

	msg, err := tgb.SendMessage(ctx, *chatID, *text,
		tg.ChatParseMode(tg.ParseMode(*parseMode)),
		tg.ChatMessageThreadID(*messageThreadID),
		tg.ChatDisableWebPagePreview(*disableWebPagePreview),
		tg.ChatDisableNotification(*disableNotification),
		tg.ChatProtectContent(*protectContent),
	)
	if err != nil {
		logFatal(err.Error())
	}

	log.Info("Success send message",
		slog.Int64("chat_id", *chatID),
		slog.Int("message_id", msg.MessageID),
	)
}
