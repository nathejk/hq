package payment

import (
	"fmt"
	"log"

	"github.com/jrgensen/cqrs"
	"github.com/jrgensen/stream"
	"github.com/jrgensen/stream/subject"
	"github.com/nathejk/shared-go/messages"
	"github.com/nathejk/shared-go/types"
)

type consumer struct {
	w cqrs.Writer
}

func (c *consumer) Consumes() (subjs []stream.Subject) {
	return []stream.Subject{
		//subject.FromStr("monolith:nathejk_team"),
		//subject.FromStr("nathejk"),
		subject.FromStr("NATHEJK.*.payment.*.requested"),
		subject.FromStr("NATHEJK.*.payment.*.reserved"),
		subject.FromStr("NATHEJK.*.payment.*.received"),
	}
}

func (c *consumer) HandleMessage(msg stream.Message) error {
	switch true {
	case msg.Subject().Match("NATHEJK.*.payment.*.requested"):
		var body messages.NathejkPaymentRequested
		if err := msg.Body(&body); err != nil {
			return err
		}
		if body.Reference == "" {
			return nil
		}
		sql := fmt.Sprintf("INSERT INTO payment SET reference=%q, receiptEmail=%q, returnUrl=%q, year=\"%d\", currency=%q, amount=%d, method=%q, createdAt=%q, changedAt=%q, status=%q, orderForeignKey=%q, orderType=%q, operations=JSON_ARRAY(JSON_OBJECT('type','requested','amount',%d,'time',%q)) ON DUPLICATE KEY UPDATE receiptEmail=VALUES(receiptEmail), returnUrl=VALUES(returnUrl), year=VALUES(year), currency=VALUES(currency), amount=VALUES(amount), method=VALUES(method), status=VALUES(status), orderForeignKey=VALUES(orderForeignKey), orderType=VALUES(orderType), operations=VALUES(operations)", body.Reference, body.ReceiptEmail, body.ReturnUrl, msg.Time().Year(), body.Currency, body.Amount, body.Method, msg.Time(), msg.Time(), types.PaymentStatusRequested, body.OrderForeignKey, body.OrderType, body.Amount, msg.Time())
		return c.w.Consume(sql)

	case msg.Subject().Match("NATHEJK.*.payment.*.reserved"):
		var body messages.NathejkPaymentReserved
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.w.Consume(fmt.Sprintf("UPDATE payment SET status=%q, changedAt=%q, operations=JSON_ARRAY_APPEND(operations,'$',JSON_OBJECT('type','reserved','amount',%d,'time',%q)) WHERE reference=%q", types.PaymentStatusReserved, msg.Time(), body.Amount, msg.Time(), body.Reference))

	case msg.Subject().Match("NATHEJK.*.payment.*.received"):
		var body messages.NathejkPaymentReceived
		if err := msg.Body(&body); err != nil {
			return err
		}
		return c.w.Consume(fmt.Sprintf("UPDATE payment SET status=%q, changedAt=%q, operations=JSON_ARRAY_APPEND(operations,'$',JSON_OBJECT('type','received','amount',%d,'time',%q)) WHERE reference=%q", types.PaymentStatusReceived, msg.Time(), body.Amount, msg.Time(), body.Reference))

	default:
		log.Printf("Unhandled message %q", msg.Subject().Subject())
	}
	return nil
}
