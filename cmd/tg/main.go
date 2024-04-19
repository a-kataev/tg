//nolint:wrapcheck
package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"

	"github.com/a-kataev/tg"
	"github.com/a-kataev/tg/cmd/tg/internal/cmd"
)

type flags struct {
	token                 string
	chatID                int64
	text                  string
	parseMode             string
	messageID             int64
	messageThreadID       int64
	disableWebPagePreview bool
	disableNotification   bool
	protectContent        bool
}

func (f *flags) tokenFormEnv() {
	if f.token == "" {
		f.token = os.Getenv("TG_TOKEN")
	}
}

func (f *flags) textFromPipe() error {
	if f.text == "-" {
		stdin, err := io.ReadAll(io.LimitReader(os.Stdin, int64(tg.MaxTextSize)))
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
		}

		f.text = string(stdin)
	}

	return nil
}

func (f *flags) rootFlags() func(*flag.FlagSet) {
	return func(fset *flag.FlagSet) {
		fset.StringVar(&f.token, "token", "", "bot token")
	}
}

func (f *flags) sendFlags() func(*flag.FlagSet) {
	return func(fset *flag.FlagSet) {
		fset.Int64Var(&f.chatID, "chat-id", 0, "chat id")
		fset.StringVar(&f.text, "text", "", "text (use - for read pipe)")
		fset.StringVar(&f.parseMode, "parse-mode", "Markdown", "parse mode")
		fset.Int64Var(&f.messageThreadID, "message-thread-id", 0, "message thread id")
		fset.BoolVar(&f.disableWebPagePreview, "disable-web-page-preview", false, "disable web page preview")
		fset.BoolVar(&f.disableNotification, "disable-notification", false, "disable notification")
		fset.BoolVar(&f.protectContent, "protect-content", false, "protect content")
	}
}

func (f *flags) sendRun(ctx context.Context, log *slog.Logger) func() error {
	return func() error {
		f.tokenFormEnv()

		if err := f.textFromPipe(); err != nil {
			return err
		}

		client, err := tg.NewClient(f.token)
		if err != nil {
			return err
		}

		msg, err := client.SendMessage(ctx, f.chatID, f.text,
			tg.ParseModeSendOption(tg.ParseMode(f.parseMode)),
			tg.MessageThreadIDSendOption(f.messageThreadID),
			tg.DisableWebPagePreviewSendOption(f.disableWebPagePreview),
			tg.DisableNotificationSendOption(f.disableNotification),
			tg.ProtectContentSendOption(f.protectContent),
		)
		if err != nil {
			return err
		}

		log.Info("Success send message",
			slog.Int64("chat_id", f.chatID),
			slog.Any("message_id", msg.MessageID),
		)

		return nil
	}
}

func (f *flags) editFlags() func(*flag.FlagSet) {
	return func(fset *flag.FlagSet) {
		fset.Int64Var(&f.chatID, "chat-id", 0, "chat id")
		fset.StringVar(&f.text, "text", "", "text (use - for read pipe)")
		fset.StringVar(&f.parseMode, "parse-mode", "Markdown", "parse mode")
		fset.Int64Var(&f.messageID, "message-id", 0, "message id")
	}
}

func (f *flags) editRun(ctx context.Context, log *slog.Logger) func() error {
	return func() error {
		f.tokenFormEnv()

		if err := f.textFromPipe(); err != nil {
			return err
		}

		client, err := tg.NewClient(f.token)
		if err != nil {
			return err
		}

		msg, err := client.EditMessage(ctx, f.chatID, f.messageID, f.text,
			tg.ParseModeEditOption(tg.ParseMode(f.parseMode)),
		)
		if err != nil {
			return err
		}

		log.Info("Success edit message",
			slog.Int64("chat_id", f.chatID),
			slog.Any("message_id", msg.MessageID),
		)

		return nil
	}
}

func (f *flags) deleteFlags() func(*flag.FlagSet) {
	return func(fset *flag.FlagSet) {
		fset.Int64Var(&f.chatID, "chat-id", 0, "chat id")
		fset.Int64Var(&f.messageID, "message-id", 0, "message id")
	}
}

func (f *flags) deleteRun(ctx context.Context, log *slog.Logger) func() error {
	return func() error {
		f.tokenFormEnv()

		client, err := tg.NewClient(f.token)
		if err != nil {
			return err
		}

		_, err = client.DeleteMessage(ctx, f.chatID, f.messageID)
		if err != nil {
			return err
		}

		log.Info("Success delete message",
			slog.Int64("chat_id", f.chatID),
			slog.Any("message_id", f.messageID),
		)

		return nil
	}
}

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := cmd.New()

	flags := new(flags)

	app.Root("tg", flags.rootFlags())
	app.Command("send", "send message", flags.sendFlags(), flags.sendRun(ctx, log))
	app.Command("edit", "edit message", flags.editFlags(), flags.editRun(ctx, log))
	app.Command("delete", "delete message", flags.deleteFlags(), flags.deleteRun(ctx, log))

	app.Run()
}
