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

func logFatal(log *slog.Logger, msg string) {
	log.Error(msg)

	os.Exit(1)
}

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

	err := fset.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(flag.ErrHelp, err) {
			fmt.Fprintln(os.Stderr, "tg - telegram bot 🤖 send message ✉️")
			fset.SetOutput(os.Stderr)
			fset.Usage()

			os.Exit(1)
		}

		logFatal(log, err.Error())
	}

	if *token == "" {
		*token = os.Getenv("TG_TOKEN")
	}

	if *text == "-" {
		stdin, err := io.ReadAll(io.LimitReader(os.Stdin, int64(tg.MaxTextSize)))
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logFatal(log, err.Error())
			}
		}

		*text = string(stdin)
	}

	client, err := tg.NewClient(*token)
	if err != nil {
		logFatal(log, err.Error())
	}

	ctx := context.Background()

	bot, err := client.GetMe(ctx)
	if err != nil {
		logFatal(log, err.Error())
	}

	log = log.With(slog.String("bot_name", bot.UserName))

	msg, err := client.SendMessage(ctx, *chatID, *text,
		tg.ParseModeSendOption(tg.ParseMode(*parseMode)),
		tg.MessageThreadIDSendOption(*messageThreadID),
		tg.DisableWebPagePreviewSendOption(*disableWebPagePreview),
		tg.DisableNotificationSendOption(*disableNotification),
		tg.ProtectContentSendOption(*protectContent),
	)
	if err != nil {
		logFatal(log, err.Error())
	}

	log.Info("Success send message",
		slog.Int64("chat_id", *chatID),
		slog.Any("message_id", msg.MessageID),
	)
}
