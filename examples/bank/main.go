// Command bank models accounts with typed errors, interfaces, and deferred
// audit logging.
package main

import "errors"

var ErrInsufficient = errors.New("insufficient funds")

type AccountError struct {
	ID  string
	Err error
}

func (e *AccountError) Error() string { return e.ID + ": " + e.Err.Error() }
func (e *AccountError) Unwrap() error { return e.Err }

type Account struct {
	ID      string
	Balance int64
	history []string
}

func (a *Account) Deposit(n int64) {
	a.Balance += n
	a.history = append(a.history, "deposit")
}

func (a *Account) Withdraw(n int64) error {
	if n > a.Balance {
		return &AccountError{a.ID, ErrInsufficient}
	}
	a.Balance -= n
	a.history = append(a.history, "withdraw")
	return nil
}

type Ledger interface {
	Record(from, to string, amount int64)
}

type memLedger struct{ lines []string }

func (m *memLedger) Record(from, to string, amount int64) {
	m.lines = append(m.lines, from+"->"+to)
}

func Transfer(l Ledger, from, to *Account, amount int64) (err error) {
	defer func() {
		if err == nil {
			l.Record(from.ID, to.ID, amount)
		}
	}()
	if err := from.Withdraw(amount); err != nil {
		return err
	}
	to.Deposit(amount)
	return nil
}

func main() {
	alice := &Account{ID: "alice", Balance: 100}
	bob := &Account{ID: "bob"}
	ledger := &memLedger{}
	for _, amt := range []int64{30, 50, 40} {
		if err := Transfer(ledger, alice, bob, amt); err != nil {
			println("transfer failed:", err.Error(), errors.Is(err, ErrInsufficient))
			continue
		}
		println("moved", amt)
	}
	println(alice.Balance, bob.Balance, len(ledger.lines), len(alice.history))
}
